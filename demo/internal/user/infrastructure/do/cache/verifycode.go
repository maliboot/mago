package cache

import "time"

// VerifyCodeRecord 是一次性验证码的内存快照（按通用分类 cache 落位，只放字段与语义）。
// ⚠️ demo 专用：生产请把验证码落到 Redis / 数据库（多副本共享 + 重启不丢 + 可限次），
// 详见 repository.VerifyCode 的说明。
type VerifyCodeRecord struct {
	// Code 是 6 位数字验证码。
	Code string
	// ExpireAt 是过期时间，过了就不再接受。
	ExpireAt time.Time
}
