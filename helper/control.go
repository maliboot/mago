package helper

import (
	"math/rand"
	"time"
)

// BackoffRetry 指数退避 + 抖动
func BackoffRetry(attempt int) time.Duration {
	base := time.Second * time.Duration(1<<attempt) // 1,2,4,8...
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	return base + jitter
}
