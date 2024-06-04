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
	return NewConvertorVO[T](err, data, 0)
}

func NewLowerCamelVO[T any](err error, data interface{}) *VO[T] {
	return NewConvertorVO[T](err, data, 1)
}

func NewSnakeVO[T any](err error, data interface{}) *VO[T] {
	return NewConvertorVO[T](err, data, 2)
}

func NewZeroVO[T any]() *VO[T] {
	vo := &VO[T]{
		IsZero: true,
	}

	return vo
}

func NewConvertorVO[T any](err error, data interface{}, convertType int) *VO[T] {
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
		switch convertType {
		case 1:
			vo.Data, _ = helper.LowerCamelConvertor[T](data)
			break
		case 2:
			vo.Data, _ = helper.SnakeConvertor[T](data)
			break
		default:
			vo.Data, _ = helper.Convertor[T](data)
		}
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
