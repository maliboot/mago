package archiveq

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	eghelper "github.com/maliboot/mago/helper"
	"github.com/maliboot/mago/queuex/logq"
)

//go:embed archive_sqlite.sql
var sqliteScheme string

type ArchiverMessage struct {
	ID        int    `json:"id"`
	TaskID    string `json:"task_id"`
	Name      string `json:"name"`
	Body      []byte `json:"body"`
	TaskGroup string `json:"task_group"`
	Result    int    `json:"result"` // 0失败 1成功
	ResultTxt string `json:"resultTxt"`
	Ext       string `json:"ext"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
}

type Archiver struct {
	ctx      context.Context
	cancel   context.CancelFunc
	cacheDir string

	failure            *sql.DB
	failureWriteLocker sync.RWMutex
	success            *sql.DB
	successWriteLocker sync.RWMutex

	tickerOnce sync.Once
}

func NewArchiver(ctx context.Context, cacheDir string) *Archiver {
	ctx, cancel := context.WithCancel(ctx)
	return &Archiver{
		ctx:      ctx,
		cancel:   cancel,
		cacheDir: cacheDir,
	}
}

func (a *Archiver) Start() error {
	var err error
	a.success, err = initDB(a.cacheDir, "success")
	if err != nil {
		return err
	}
	a.failure, err = initDB(a.cacheDir, "failure")
	if err != nil {
		return err
	}

	a.tickerOnce.Do(func() {
		eghelper.Cron(a.ctx, "消息队列归档-清理器", time.Minute*30, func(ctx context.Context) {
			go func() {
				if _, err := a.batchDelete(ctx, true, 3); err != nil {
					logq.Log.Errorf("消息队列归档-清理旧成功消息失败: %v", err)
				}
			}()
			if _, err := a.batchDelete(ctx, false, 7); err != nil {
				logq.Log.Errorf("消息队列归档-清理旧异常消息失败: %v", err)
			}
		})
	})

	return nil
}

func (a *Archiver) Stop() {
	if a.success != nil {
		_ = a.success.Close()
	}
	if a.failure != nil {
		_ = a.failure.Close()
	}
	a.cancel()
}

func (a *Archiver) Add(isSuccess bool, messages []*ArchiverMessage) error {
	if a.success == nil {
		return fmt.Errorf("QueueArchiver还未初始化完成")
	}
	var db *sql.DB
	if isSuccess {
		db = a.success
		a.successWriteLocker.Lock()
		defer a.successWriteLocker.Unlock()
	} else {
		db = a.failure
		a.failureWriteLocker.Lock()
		defer a.failureWriteLocker.Unlock()
	}

	return a.batchInsert(context.Background(), db, messages)
}

func (a *Archiver) Get(ctx context.Context, isSuccess bool, name, taskGroup string, page, pageSize int) ([]*ArchiverMessage, error) {
	if a.success == nil {
		return nil, fmt.Errorf("QueueArchiver还未初始化完成")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}

	var db *sql.DB
	if isSuccess {
		db = a.success
	} else {
		db = a.failure
	}

	dataQuery := "SELECT id, task_id, created, updated, name, body, task_group, result, result_txt, ext FROM archive_message WHERE 1=1"
	dataArgs := make([]any, 0)

	if name != "" {
		dataQuery += " AND `name` = ?"
		dataArgs = append(dataArgs, name)
	}
	if taskGroup != "" {
		dataQuery += " AND `task_group` = ?"
		dataArgs = append(dataArgs, taskGroup)
	}

	dataQuery += `
        ORDER BY created ASC
        LIMIT ? OFFSET ?
    `
	dataArgs = append(dataArgs, pageSize, (page-1)*pageSize)

	rows, err := db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("队列归档-分页查询数据失败: %w", err)
	}
	defer rows.Close()

	var messages []*ArchiverMessage
	for rows.Next() {
		var m ArchiverMessage

		if err := rows.Scan(&m.ID, &m.TaskID, &m.Created, &m.Updated, &m.Name, &m.Body, &m.TaskGroup, &m.Result, &m.ResultTxt, &m.Ext); err != nil {
			return nil, fmt.Errorf("扫描行失败: %w", err)
		}

		messages = append(messages, &m)
	}

	return messages, nil
}

func (a *Archiver) batchDelete(ctx context.Context, isSuccess bool, days int) (int64, error) {
	if days <= 0 {
		days = 3
	}
	var db *sql.DB
	var msgResult = 1
	if isSuccess {
		db = a.success
		a.successWriteLocker.Lock()
		defer a.successWriteLocker.Unlock()
	} else {
		msgResult = 0
		db = a.failure
		a.failureWriteLocker.Lock()
		defer a.failureWriteLocker.Unlock()
	}

	const batchSize = 2000 // 每次最多删除 5000 条，防止大表长时间锁

	var totalDeleted int64
	for {
		sqlRet, err := db.ExecContext(ctx, fmt.Sprintf(
			`
DELETE FROM archive_message 
WHERE result = %d 
  AND created < datetime('now', '-%d day')
  AND rowid IN (
      SELECT rowid 
      FROM archive_message 
      WHERE result = %d 
        AND created < datetime('now', '-%d day')
      ORDER BY created ASC, rowid ASC
      LIMIT %d
  );
`,
			msgResult, days, msgResult, days, batchSize,
		))

		if err != nil {
			return totalDeleted, fmt.Errorf("归档队列消息-删除旧消息失败: %w", err)
		}

		deleted, err := sqlRet.RowsAffected()
		if err != nil {
			return totalDeleted, err
		}

		totalDeleted += deleted

		if deleted < int64(batchSize) {
			break // 本批次已删完，退出循环
		}
	}

	return totalDeleted, nil
}

func (a *Archiver) batchInsert(ctx context.Context, db *sql.DB, msgs []*ArchiverMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	const batchSize = 200
	for i := 0; i < len(msgs); i += batchSize {
		end := i + batchSize
		if end > len(msgs) {
			end = len(msgs)
		}

		batch := msgs[i:end]
		placeholders := make([]string, len(batch))
		args := make([]any, 0, len(batch)*6)

		for j, m := range batch {
			placeholders[j] = "(?, ?, ?, ?, ?, ?, ?, ?, ?)"
			args = append(args, m.TaskID, m.TaskGroup, m.Created, m.Updated, m.Name, m.Body, m.Result, m.ResultTxt, m.Ext)
		}

		query := `INSERT INTO archive_message 
                  (task_id, task_group, created, updated, name, body, result, result_txt, ext) 
                  VALUES ` + strings.Join(placeholders, ",")

		_, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
	}
	return nil
}

func initDB(cacheDir string, name string) (*sql.DB, error) {
	dbName := path.Join(cacheDir, fmt.Sprintf("archiver_%s.db?_journal=WAL&_temp_store=MEMORY&_cache_size=-16000&_busy_timeout=5000", name))
	db, err := eghelper.SqliteOpen(dbName)
	if err != nil {
		return nil, fmt.Errorf("dbArchiverInitDBErr.0: %v", err)
	}

	exists, err := eghelper.IsSqliteTableExists(db, "archive_message")
	if err != nil {
		return nil, fmt.Errorf("dbArchiverInitDBErr.1: %v", err)
	}
	if !exists {
		if _, err = db.Exec(sqliteScheme); err != nil {
			return nil, fmt.Errorf("dbArchiverInitDBErr.2: %v", err)
		}
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}
