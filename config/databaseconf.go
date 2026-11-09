package config

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/maliboot/mago/logx"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	rawmysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type DataBaseConf struct {
	Type          string `yaml:"type"`
	Dsn           string `yaml:"dsn"`
	Net           string `yaml:"net"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	DBName        string `yaml:"db"`
	TablePrefix   string `yaml:"table_prefix"`
	SingularTable bool   `yaml:"singular_table"`
	NoLowerCase   bool   `yaml:"no_lower_case"`
	BackupDir     string `yaml:"backup_dir"`

	dbConnector     *dbConnector
	dbConnectorOnce sync.Once
	Ctx             any
}

type dbConnector struct {
	ins *gorm.DB
	err error

	readerIns        *gorm.DB
	writerIns        *gorm.DB
	readerWriterOnce sync.Once
	readerWriterErr  error
}

func (dc *DataBaseConf) initDbConnector() {
	if dc.dbConnector == nil {
		dc.dbConnector = &dbConnector{}
	}
	if dc.Type == "" {
		dc.Type = "mysql"
	}

	var dialector gorm.Dialector
	switch dc.Type {
	case "mysql":
		if dc.Dsn != "" {
			dialector = mysql.New(mysql.Config{DSN: dc.Dsn})
		} else {
			rawConnector, err := rawmysql.NewConnector(&rawmysql.Config{
				User:                 dc.Username,
				Passwd:               dc.Password,
				Net:                  dc.Net,
				Addr:                 fmt.Sprintf("%s:%d", dc.Host, dc.Port),
				DBName:               dc.DBName,
				ParseTime:            true,
				Loc:                  time.Local,
				AllowNativePasswords: true,
				TLSConfig:            "preferred",
			})
			if err != nil {
				dc.dbConnector.err = err
				return
			}
			dialector = mysql.New(mysql.Config{Conn: sql.OpenDB(rawConnector)})
		}
		break
	case "sqlite":
		if dc.Dsn == "" {
			dc.Dsn = "file::memory:?cache=shared"
		}

		if strings.HasSuffix(dc.Dsn, ".db") {
			sqliteDir := path.Dir(dc.Dsn)
			if _, err := os.Stat(sqliteDir); err != nil {
				err = os.Mkdir(sqliteDir, 0755)
				if err != nil {
					panic(fmt.Errorf("创建数据库目录[%s]失败:[%+v]", sqliteDir, err))
				}
			}
		}
		dialector = sqlite.Open(dc.Dsn)
		break
	default:
		dc.dbConnector.err = errors.New("不支持的数据库类型")
		return
	}

	dc.dbConnectorOnce.Do(func() {
		dc.dbConnector.ins, dc.dbConnector.err = gorm.Open(dialector, &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   dc.TablePrefix,
				SingularTable: dc.SingularTable,
				NoLowerCase:   dc.NoLowerCase,
			},
			Logger: logx.NewGormLogger(),
		})
	})
}

func (dc *DataBaseConf) GetDB() (*gorm.DB, error) {
	dc.initDbConnector()
	return dc.dbConnector.ins, dc.dbConnector.err
}

func (dc *DataBaseConf) GetDBReaderWriter() (*gorm.DB, *gorm.DB, error) {
	dc.initDbConnector()
	// 读写分离
	dc.dbConnector.readerWriterOnce.Do(dc.readerWriterInit)
	return dc.dbConnector.readerIns, dc.dbConnector.writerIns, dc.dbConnector.err
}

func (dc *DataBaseConf) GetSqliteDB(dsnArgs string) (*gorm.DB, error) {
	dsn := dc.Dsn
	if dsnArgs != "" {
		if strings.HasSuffix(dsn, ".db") {
			dsn = dsn + "?"
		}
		dsn = dsn + dsnArgs
	}

	return gorm.Open(sqlite.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   dc.TablePrefix,
			SingularTable: dc.SingularTable,
			NoLowerCase:   dc.NoLowerCase,
		},
		Logger: logx.NewGormLogger(),
	})
}

func (dc *DataBaseConf) SqliteWalInit(db *gorm.DB) {
	// sqlite PRAGMA 优化
	if _, ok := db.Dialector.(*sqlite.Dialector); !ok {
		return
	}
	stmts := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA temp_store = MEMORY;",
		"PRAGMA cache_size = -64000;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA wal_autocheckpoint = 1000;",
	}
	for _, s := range stmts {
		if tx := db.Exec(s); tx.Error != nil {
			hlog.Errorf("db[%s] Failed to set WAL: %v", db.Name(), tx.Error)
			break
		}
	}
}

func (dc *DataBaseConf) readerWriterInit() {
	if dc.dbConnector.ins == nil || dc.dbConnector.err != nil {
		hlog.Fatalf("默认数据连接初始化失败: %v", dc.dbConnector.err)
	}

	// 读写分离
	if _, ok := dc.dbConnector.ins.Dialector.(*sqlite.Dialector); ok {
		writerDsn := "_cache=shared&_journal=WAL&_synchronous=NORMAL&_temp_store=MEMORY&_cache_size=-64000&_busy_timeout=5000"
		// 写连接
		dc.dbConnector.writerIns, dc.dbConnector.readerWriterErr = dc.GetSqliteDB(writerDsn)
		if dc.dbConnector.readerWriterErr != nil {
			hlog.Fatalf("db[%s] Failed to get writer: %v", dc.dbConnector.ins.Name(), dc.dbConnector.readerWriterErr)
		}
		// 写库建议保持单连接串行写
		sqliteWDb, _ := dc.dbConnector.writerIns.DB()
		sqliteWDb.SetMaxOpenConns(1)
		sqliteWDb.SetMaxIdleConns(1)
		// 读连接
		readerDsn := writerDsn
		dc.dbConnector.readerIns, dc.dbConnector.readerWriterErr = dc.GetSqliteDB(readerDsn)
		if dc.dbConnector.readerWriterErr != nil {
			hlog.Fatalf("db[%s] Failed to get reader: %v", dc.dbConnector.ins.Name(), dc.dbConnector.readerWriterErr)
		}
		sqliteRDb, _ := dc.dbConnector.readerIns.DB()
		sqliteRDb.SetMaxOpenConns(15)
		sqliteRDb.SetMaxIdleConns(10)
		sqliteRDb.SetConnMaxLifetime(10 * time.Minute)

		// 有时候会不成功，再执行一次关键 PRAGMA，确保全局生效
		dc.SqliteWalInit(dc.dbConnector.writerIns)
		// 定时checkpoint
		dc.sqliteWalCheckpoint()
	} else {
		// todo 目前只支持sqlite
		dc.dbConnector.readerIns = dc.dbConnector.ins
		dc.dbConnector.writerIns = dc.dbConnector.ins
	}
}

func (dc *DataBaseConf) sqliteWalCheckpoint() {
	if dc.dbConnector.readerWriterErr != nil || dc.dbConnector.writerIns == nil {
		return
	}
	if _, ok := dc.dbConnector.writerIns.Dialector.(*sqlite.Dialector); !ok {
		return
	}
	go func() {
		for {
			now := time.Now()

			// ---------- 1️⃣ 执行 wal_checkpoint 每 5 分钟 ----------
			if tx := dc.dbConnector.writerIns.Exec("PRAGMA wal_checkpoint(TRUNCATE);"); tx.Error != nil {
				hlog.Errorf("[sqlite-maint] checkpoint err: %v", tx.Error)
			} else {
				hlog.Debugf("[sqlite-maint] checkpoint done at %s", now.Format(time.TimeOnly))
			}

			// ---------- 2️⃣ 判断是否到凌晨 2 点 ----------
			// 触发条件：当前时间在 [02:00, 02:05) 区间内，只执行一次
			if now.Hour() == 2 && now.Minute() < 5 {
				if tx := dc.dbConnector.writerIns.Exec("VACUUM;"); tx.Error != nil {
					hlog.Errorf("[sqlite-maint] vacuum err: %v", tx.Error)
				} else {
					hlog.Debugf("[sqlite-maint] vacuum done at %s", now.Format(time.TimeOnly))
				}
				// 避免同一分钟重复执行
				time.Sleep(6 * time.Minute)
			}

			// ---------- 3️⃣ 休眠 5 分钟后继续 ----------
			time.Sleep(5 * time.Minute)
		}
	}()
}
