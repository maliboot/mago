package mago

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/gorm"

	"github.com/maliboot/mago/config"
)

type Repository[D DataObject] interface {
	Error() error
	DB(do D) *gorm.DB
	DefaultDB() *gorm.DB
	DefaultDO() D
	Find(id int) *D
	First(where [][]string, columns []string) *D
	Last(where [][]string, columns []string) *D
	Get(where [][]string, columns []string, limit int) *[]D
	Paginate(where [][]string, columns []string, orderByRaw string, page int, pageSize int) *PageVO[D]
	where(tx *gorm.DB, where [][]string) *gorm.DB
	Create(do *D) int64
	Update(id int, do *D) int64
	Delete(ids []int) int64
}

type AbstractRepository[D DataObject] struct {
	Conf *config.Conf
	Err  error
}

// txKey 是事务句柄在 ctx 中的键：私有空结构体，避免与其他包写入的 ctx 值冲突。
type txKey struct{}

// WithTx 把事务句柄放进 ctx；只在 WithTransaction（或自建事务）内部调用。
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromCtx 优先返回 ctx 中的事务句柄，没有则回退到 fallback（仓储的默认连接）。
// 事务能跨仓储共享，靠的就是同一个包级 txKey。
func TxFromCtx(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}

	return fallback
}

// WithTransaction 在单个事务内执行 fn：事务句柄经 ctx 传给 fn，
// fn 内的仓储操作（用 TxDB 取连接）自动进入同一事务；fn 返回错误即整体回滚。
// 事务边界由 application 决定（README §14.1），本方法只提供事务基础设施。
func (d *AbstractRepository[D]) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	var do D

	db := d.DB(do)
	if d.Err != nil {
		return d.Err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(WithTx(ctx, tx))
	})
}

// TxDB 返回本次调用应使用的连接：ctx 里带事务就用事务，否则用 DO 配置的默认连接；
// 返回值已 WithContext(ctx)，超时与取消继续向下透传。
// 事务内的读写一律走它，直接用 DB(do) 会绕过事务。
func (d *AbstractRepository[D]) TxDB(ctx context.Context, do D) *gorm.DB {
	return TxFromCtx(ctx, d.DB(do)).WithContext(ctx)
}

func (d *AbstractRepository[D]) Error() error {
	return d.Err
}

func (d *AbstractRepository[D]) resetErr(err error) {
	d.Err = nil
	if err == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}

	d.Err = err
}

func (d *AbstractRepository[D]) DB(do D) *gorm.DB {
	var db *gorm.DB
	dbConf, ok := d.Conf.Databases[do.DatabaseName()]
	if !ok {
		d.Err = errors.New("database conf not found")
		hlog.Fatalf("数据库配置异常")
	}
	db, d.Err = dbConf.GetDB()
	return db
}

func (d *AbstractRepository[D]) DefaultDB() *gorm.DB {
	var db *gorm.DB
	var do D
	dbConf, ok := d.Conf.Databases[do.DatabaseName()]
	if !ok {
		d.Err = errors.New("database conf not found")
		hlog.Fatalf("数据库配置异常")
	}
	db, d.Err = dbConf.GetDB()
	return db
}

func (d *AbstractRepository[D]) DefaultDO() D {
	var do D
	return do
}

func (d *AbstractRepository[D]) Find(id int) *D {
	if id == 0 {
		d.resetErr(gorm.ErrRecordNotFound)
		return nil
	}
	var do D
	tx := d.DB(do).Where("id=?", id).First(&do)
	d.resetErr(tx.Error)
	if tx.RowsAffected == 0 {
		return nil
	}
	return &do
}

func (d *AbstractRepository[D]) Create(do *D) int64 {
	tx := d.DB(*do).Create(do)
	d.resetErr(tx.Error)
	return tx.RowsAffected
}

func (d *AbstractRepository[D]) Update(id int, do *D) int64 {
	if id == 0 {
		return 0
	}
	var ds D
	tx := d.DB(*do).Model(&ds).Where((*do).PrimaryKey()+" = ?", id).Updates(do)
	d.resetErr(tx.Error)
	return tx.RowsAffected
}

func (d *AbstractRepository[D]) Delete(ids []int) int64 {
	// todo: filter deleted data
	ids = slices.DeleteFunc(ids, func(i int) bool {
		return i == 0
	})
	if len(ids) == 0 {
		return 0
	}
	var do D
	tx := d.DB(do).Delete(&do, ids)
	d.resetErr(tx.Error)
	return tx.RowsAffected
}

func (d *AbstractRepository[D]) First(where [][]string, columns []string) *D {
	var do D
	tx := d.where(d.DB(do).Select(columns), where).First(&do)
	d.resetErr(tx.Error)
	if tx.RowsAffected == 0 {
		return nil
	}
	return &do
}

func (d *AbstractRepository[D]) Last(where [][]string, columns []string) *D {
	var do D
	tx := d.where(d.DB(do).Select(columns), where).Last(&do)
	d.resetErr(tx.Error)
	if tx.RowsAffected == 0 {
		return nil
	}
	return &do
}

func (d *AbstractRepository[D]) Get(where [][]string, columns []string, limit int) *[]D {
	var result = make([]D, 0)
	var ds D
	tx := d.where(d.DB(ds).Select(columns), where).Limit(limit).Find(&result)
	d.resetErr(tx.Error)
	return &result
}

func (d *AbstractRepository[D]) GetByOrder(where [][]string, columns []string, limit int, order string) *[]D {
	var result = make([]D, 0)
	var ds D
	tx := d.where(d.DB(ds).Select(columns), where).Order(order).Limit(limit).Find(&result)
	d.resetErr(tx.Error)
	return &result
}

// Paginate 分页
func (d *AbstractRepository[D]) Paginate(pageQry *AbstractPageQuery) *PageVO[D] {
	var do D
	dbIns := d.DB(do).Model(&do)
	if pageQry.PageSize == 0 {
		pageQry.PageSize = 10
	}
	if pageQry.PageIndex == 0 {
		pageQry.PageIndex = 1
	}
	if pageQry.OrderByRaw == "" {
		pageQry.OrderByRaw = do.PrimaryKey() + " desc"
	}

	if pageQry.Columns != nil && len(pageQry.Columns) != 0 {
		dbIns = dbIns.Select(pageQry.Columns)
	}
	if pageQry.Where != nil && len(pageQry.Where) != 0 {
		dbIns = d.where(dbIns, pageQry.Where)
	}
	if pageQry.OrderByRaw != "" {
		dbIns = dbIns.Order(pageQry.OrderByRaw)
	}
	var rowCount int64
	dbIns.Count(&rowCount)

	rows := make([]D, 0)
	tx := dbIns.Scopes(Pagination(pageQry.PageIndex, pageQry.PageSize)).Find(&rows)
	d.resetErr(tx.Error)
	if d.Err != nil && tx.RowsAffected == 0 {
		return &PageVO[D]{
			PageIndex:  pageQry.PageIndex,
			PageSize:   pageQry.PageSize,
			TotalCount: 0,
			Items:      &rows,
		}
	}

	return &PageVO[D]{
		PageIndex:  pageQry.PageIndex,
		PageSize:   pageQry.PageSize,
		TotalCount: int(rowCount),
		TotalPage:  int(math.Ceil(float64(rowCount) / float64(pageQry.PageSize))),
		Items:      &rows,
	}
}

func (d *AbstractRepository[D]) Where(tx *gorm.DB, where [][]string) *gorm.DB {
	return d.where(tx, where)
}

// 组合条件
func (d *AbstractRepository[D]) where(tx *gorm.DB, where [][]string) *gorm.DB {
	if where == nil || len(where) == 0 {
		return tx
	}
	for _, v := range where {
		var query string
		var queryArgs []interface{}
		vLen := len(v)
		if vLen < 2 {
			continue
		}
		if vLen == 2 {
			query = fmt.Sprintf("%s = ?", v[0])
			queryArgs = append(queryArgs, v[1])
		}
		if vLen > 2 {
			var queryArg interface{}
			switch strings.ToUpper(v[1]) {
			case "RAW":
				query = v[0]
				break
			case "IN":
				query = fmt.Sprintf("%s %s ?", v[0], v[1])
				queryArg = strings.Split(v[2], ",")
				break
			case "NOT IN":
				query = fmt.Sprintf("%s %s ?", v[0], v[1])
				queryArg = strings.Split(v[2], ",")
				break
			default:
				query = fmt.Sprintf("%s %s ?", v[0], v[1])
				queryArg = v[2]
			}
			queryArgs = append(queryArgs, queryArg)
		}
		if vLen == 4 && strings.ToUpper(v[3]) == "OR" {
			tx = tx.Not(query, queryArgs...)
		} else {
			tx = tx.Where(query, queryArgs...)
		}
	}
	return tx
}
