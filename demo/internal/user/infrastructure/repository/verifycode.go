package repository

import (
	"context"
	"sync"
	"time"

	cachedo "tests/internal/user/infrastructure/do/cache"
)

// VerifyCode 是邮箱验证码的本机内存实现。
//
// ⚠️ 仅供 demo / 本地联调，禁止直接用于生产：
//  1. 单副本：状态只在本进程内存里，多副本 / 多机部署时各实例互不可见，
//     用户在 A 实例取码、请求落到 B 实例就会「验证码错误」。生产必须换成
//     Redis（推荐，天然带 TTL）或数据库实现，端口 VerifyCodeStore 不变，
//     只改组合根 conf/provider.go 的 provideVerifyCodeStore 一行（§7.4 规则 8）。
//  2. 易失：进程重启 / 崩溃后已发出的验证码全部失效，用户需重新获取。
//  3. 无防爆破：这里没有「错误次数上限 / 锁定」，生产实现必须补上，
//     否则 6 位数字码可被在线穷举。
//
// 换 Redis 时的落位建议：key 用 email 维度（如 verify_code:{email}），
// value 存 code + 错误计数，TTL 与邮件文案一致；核销用「取出即删」的原子操作
// （Lua 或 GETDEL），避免并发下同一个码被用两次。
type VerifyCode struct {
	mu    sync.Mutex
	codes map[string]cachedo.VerifyCodeRecord
}

// NewVerifyCode 非单例：验证码只经 application 的 VerifyCodeStore 端口使用，
// 由组合根注入同一个实例即可，不需要包级单例（§16.8）。
func NewVerifyCode() *VerifyCode {
	return &VerifyCode{codes: make(map[string]cachedo.VerifyCodeRecord)}
}

// Save 写入验证码，同邮箱的旧码会被覆盖（后发的码必须顶掉前一个）。
func (v *VerifyCode) Save(ctx context.Context, email, code string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	v.codes[email] = cachedo.VerifyCodeRecord{Code: code, ExpireAt: time.Now().Add(ttl)}

	return nil
}

// Consume 校验并核销验证码：成功后立即失效，避免同一个码被重复使用。
// 填错不删除（用户可能手滑），过期才清理；真实项目还要加「错误次数上限」防爆破。
func (v *VerifyCode) Consume(ctx context.Context, email, code string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	record, ok := v.codes[email]
	if !ok {
		return false, nil
	}
	if time.Now().After(record.ExpireAt) {
		delete(v.codes, email)

		return false, nil
	}
	if record.Code != code {
		return false, nil
	}

	delete(v.codes, email)

	return true, nil
}
