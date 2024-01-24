package mago

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/maliboot/mago/config"
	"gorm.io/gorm"
)

type Repository[D DataObject] interface {
	Error() error
	DB(do D) *gorm.DB
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
	db, d.Err = d.Conf.Databases[do.DatabaseName()].GetDB()
	return db
}

func (d *AbstractRepository[D]) Find(id int) *D {
	if id == 0 {
		d.resetErr(gorm.ErrRecordNotFound)
		return nil
	}
	var do = new(D)
	tx := d.DB(*do).Where("id=?", id).First(do)
	d.resetErr(tx.Error)
	if tx.RowsAffected == 0 {
		return nil
	}
	return do
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
	tx := d.DB(*do).Model(new(D)).Where((*do).PrimaryKey()+" = ?", id).Updates(do)
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
	var do = new(D)
	tx := d.DB(*do).Delete(do, ids)
	d.resetErr(tx.Error)
	return tx.RowsAffected
}

func (d *AbstractRepository[D]) First(where [][]string, columns []string) *D {
	var do = new(D)
	tx := d.where(d.DB(*do).Select(columns), where).First(do)
	d.resetErr(tx.Error)
	if tx.RowsAffected == 0 {
		return nil
	}
	return do
}

func (d *AbstractRepository[D]) Last(where [][]string, columns []string) *D {
	var do = new(D)
	tx := d.where(d.DB(*do).Select(columns), where).Last(do)
	d.resetErr(tx.Error)
	if tx.RowsAffected == 0 {
		return nil
	}
	return do
}

func (d *AbstractRepository[D]) Get(where [][]string, columns []string, limit int) *[]D {
	var result = make([]D, 0)
	tx := d.where(d.DB(*(new(D))).Select(columns), where).Limit(limit).Find(&result)
	d.resetErr(tx.Error)
	return &result
}

// Paginate 分页
func (d *AbstractRepository[D]) Paginate(pageQry *AbstractPageQuery) *PageVO[D] {
	var do = new(D)
	dbIns := d.DB(*do).Model(do)
	if pageQry.PageSize == 0 {
		pageQry.PageSize = 10
	}
	if pageQry.PageIndex == 0 {
		pageQry.PageIndex = 1
	}
	if pageQry.OrderByRaw == "" {
		pageQry.OrderByRaw = (*do).PrimaryKey() + " desc"
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
