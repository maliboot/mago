package mago

type AbstractDTO struct {
	UserId    uint   `json:"userId,omitempty" query:"__id__"`
	RequestId string `json:"requestId,omitempty" query:"X-Request-ID"`
}

func (a AbstractDTO) BindErr() interface{} {
	return NewVO[EmptyVO](NewError(ErrInvalidParams).WithMsg("请检查请求json格式、字段类型是否正确"), nil)
}
