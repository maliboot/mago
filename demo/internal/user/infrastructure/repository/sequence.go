package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	cachedo "tests/internal/user/infrastructure/do/cache"
)

// seqWidth 是当天序号的最小宽度，超出后按实际位数输出。
const seqWidth = 4

// Sequence 是编号计数器的本机内存实现：进程内计数，不落盘。
//
// 语义：Next 返回「前缀 + yyyyMMdd + 当天第几次调用」，跨天自动从 0001 重新计数。
// 边界：计数只存在内存里，进程重启后从 0001 重新开始，编号可能重复；
// 需要全局唯一时换 Redis / 数据库实现，端口保持不变（§7.4 规则 8）。
type Sequence struct {
	mu    sync.Mutex
	state cachedo.Counter
}

var (
	sequenceInstance *Sequence
	sequenceOnce     sync.Once
)

// NewSequence 返回进程内唯一的编号计数器（§16.8 单例）。
// 判断依据：计数器持有进程内唯一状态，多实例会各自计数导致重号。
// 测试需要隔离计数时直接构造 &Sequence{}，不要走这个入口。
func NewSequence() *Sequence {
	sequenceOnce.Do(func() {
		sequenceInstance = &Sequence{}
	})

	return sequenceInstance
}

// Next 发放下一个编号，如 Ym202611110001。
func (s *Sequence) Next(ctx context.Context, prefix string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	today := time.Now().Format("20060102")
	if s.state.Date != today {
		// 首次调用或跨天，重置当天计数
		s.state = cachedo.Counter{Date: today}
	}

	s.state.Value++

	return fmt.Sprintf("%s%s%0*d", prefix, today, seqWidth, s.state.Value), nil
}
