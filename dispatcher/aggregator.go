package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	eghelper "github.com/maliboot/mago/helper"
)

type AggregatorEvent[T any] struct {
	Key  string
	Data T

	Metadata map[string]string
}

type AggregatorFlushFn[T any] func(string, []*AggregatorEvent[T])

type Aggregator[T any] struct {
	name   string
	ctx    context.Context
	cancel context.CancelFunc

	mu    sync.Mutex
	cache map[string][]*AggregatorEvent[T] // 按 Key 聚合

	flushNum         int
	flushTime        time.Duration
	flushGracePeriod time.Duration // todo
	flushFn          AggregatorFlushFn[T]

	startOnce sync.Once
}

func NewAggregator[T any](ctx context.Context, name string, flushTime time.Duration, flushFn AggregatorFlushFn[T]) *Aggregator[T] {
	if flushTime == 0 {
		flushTime = 1000 * time.Millisecond
	}
	newCtx, cancel := context.WithCancel(ctx)
	ins := &Aggregator[T]{
		ctx:       newCtx,
		name:      name,
		flushTime: flushTime,
		cancel:    cancel,
		flushFn:   flushFn,
		cache:     make(map[string][]*AggregatorEvent[T]),
	}
	ins.initialize()
	return ins
}

func (a *Aggregator[T]) initialize() {
	a.SetFlushNum(0)
}

func (a *Aggregator[T]) SetFlushNum(flushNum int) *Aggregator[T] {
	if flushNum == 0 {
		flushNum = 15
	}
	a.flushNum = flushNum
	return a
}

func (a *Aggregator[T]) Add(event *AggregatorEvent[T]) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cache[event.Key] = append(a.cache[event.Key], event)

	// 数量达到阈值
	if len(a.cache[event.Key]) >= a.flushNum {
		a.flush(event.Key)
	}
}

func (a *Aggregator[T]) Start() {
	a.startOnce.Do(func() {
		eghelper.Cron(a.ctx, fmt.Sprintf("聚合器[%s]扫描服务", a.name), a.flushTime, func(ctx context.Context) {
			a.mu.Lock()
			for key := range a.cache {
				a.flush(key)
			}
			a.mu.Unlock()
		})
	})
}

func (a *Aggregator[T]) flush(key string) {
	events := a.cache[key]
	if len(events) == 0 {
		return
	}

	delete(a.cache, key)
	if a.flushFn != nil {
		a.flushFn(key, events)
	}
}
