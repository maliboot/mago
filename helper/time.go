package helper

import (
	"context"
	"fmt"
	"time"
)

func DoTimeout(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- fn(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("timedout[%dms]:%v", timeout/time.Millisecond, ctx.Err())
	}
}
