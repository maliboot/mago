package mago

import "gorm.io/gorm"

type AbstractPageQuery struct {
	PageSize   int `query:"pageSize"`
	PageIndex  int `query:"pageIndex"`
	Columns    []string
	OrderByRaw string
	Where      [][]string
}

func Pagination(page int, perPage int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page == 0 {
			page = 1
		}

		switch {
		case perPage > 100:
			perPage = 100
		case perPage <= 0:
			perPage = 10
		}

		offset := (page - 1) * perPage
		return db.Offset(offset).Limit(perPage)
	}
}
