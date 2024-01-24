package mago

type PageVO[T any] struct {
	AbstractViewObject
	PageSize   int  `json:"pageSize"`
	PageIndex  int  `json:"pageIndex"`
	TotalCount int  `json:"totalCount"`
	TotalPage  int  `json:"totalPage"`
	Items      *[]T `json:"items"`
}
