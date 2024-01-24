package mago

import "github.com/maliboot/mago/helper"

type ViewObject interface {
}

type AbstractViewObject struct {
	ViewObject `json:"-"`
}

type ViewObjectTemplate interface {
	Zero
	Error() error
	Body() interface{}
}

type VO[T any] struct {
	Data   *T
	Err    error
	IsZero bool
}

func NewVO[T any](err error, data interface{}) *VO[T] {
	vo := &VO[T]{
		Err: err,
	}
	if data == nil {
		vo.IsZero = true
		return vo
	}

	myData, ok := data.(*T)
	if ok {
		vo.Data = myData
	} else {
		vo.Data, _ = helper.Convertor[T](data)
	}

	return vo
}

func NewZeroVO[T any]() *VO[T] {
	vo := &VO[T]{
		IsZero: true,
	}

	return vo
}

func (vo *VO[T]) Error() error {
	return vo.Err
}

func (vo *VO[T]) Zero() bool {
	return vo.IsZero
}

func (vo *VO[T]) Body() interface{} {
	return vo.Data
}
