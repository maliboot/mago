package constant

// 本包放「adapter 与 application 共用的常量」：任务名、事件名、对外约定的标识等。
// 只放常量，不放结构体与逻辑（结构体属于契约，放 client/dto 或 client/vo）。

// 队列任务名（生产端 application 与消费端 adapter/mq 共用）。
const (
	// TaskUserRegistered 是「用户已注册」异步任务名。
	TaskUserRegistered = "user_registered"
)
