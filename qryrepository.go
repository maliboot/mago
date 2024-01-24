package mago

type QryRepository[D DataObject] interface {
	Repository[D]
	ListByPage(id []int)
}

type QryRepo[D DataObject] struct {
	AbstractRepository[D]
}

func (q QryRepo[D]) ListByPage(pageQry *AbstractPageQuery) *PageVO[D] {
	return q.Paginate(pageQry)
}
