package mago

import (
	"github.com/maliboot/mago/helper"
)

type CmdRepository[E Entity] interface {
	Error() error
	Find(id int) *E
	// Create 单条添加 返回: 自增ID
	Create(entity *E) uint
	Update(id int, e *E)
	Delete(id []int)
}

type CmdRepo[D DataObject, E Entity] struct {
	AbstractRepository[D]
}

func (cr *CmdRepo[D, E]) Find(id int) *E {
	d := cr.AbstractRepository.Find(id)
	if d == nil {
		return nil
	}

	var e *E
	e, cr.Err = helper.Convertor[E](d)
	return e
}

func (cr *CmdRepo[D, E]) Create(e *E) uint {
	var d *D
	d, cr.Err = helper.Convertor[D](e)
	if cr.Err != nil {
		return 0
	}

	cr.AbstractRepository.Create(d)
	return (*d).PrimaryValue()
}

func (cr *CmdRepo[D, E]) Update(id int, e *E) {
	var d *D
	d, cr.Err = helper.Convertor[D](e)
	if cr.Err != nil {
		return
	}

	cr.AbstractRepository.Update(id, d)
}

func (cr *CmdRepo[D, E]) Delete(id []int) {
	cr.AbstractRepository.Delete(id)
}
