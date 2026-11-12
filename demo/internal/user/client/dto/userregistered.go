package dto

import "github.com/maliboot/mago"

// UserRegisteredEvent 是「用户已注册」异步任务的载荷：
// 与任务名（client/constant.TaskUserRegistered）一起构成生产端与消费端的契约，改字段即改契约。
// 只放字段与 json 标签，不写解析逻辑、不引用领域对象。
type UserRegisteredEvent struct {
	// UserID 是新建用户的 ID。
	UserID uint `json:"userId"`
	// Email 是规范化后的注册邮箱。
	Email string `json:"email"`
	// At 是注册完成时间，JSON 形如 "2006-01-02 15:04:05"。
	At mago.DateTime `json:"at"`
}
