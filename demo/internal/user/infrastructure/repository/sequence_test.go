package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	cachedo "tests/internal/user/infrastructure/do/cache"
)

// today 返回当天日期串，与实现保持同一口径。
func today() string {
	return time.Now().Format("20060102")
}

func TestSequenceFormatAndIncrement(t *testing.T) {
	seq := &Sequence{}
	ctx := context.Background()

	first, err := seq.Next(ctx, "Ym")
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if want := "Ym" + today() + "0001"; first != want {
		t.Fatalf("首个编号 = %q, 期望 %q", first, want)
	}

	second, err := seq.Next(ctx, "Ym")
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if want := "Ym" + today() + "0002"; second != want {
		t.Fatalf("第二个编号 = %q, 期望 %q", second, want)
	}
}

func TestSequenceResetsEachDay(t *testing.T) {
	seq := &Sequence{}

	// 伪造「昨天已经发到 99 个」的内存状态
	seq.state = cachedo.Counter{Date: time.Now().AddDate(0, 0, -1).Format("20060102"), Value: 99}

	code, err := seq.Next(context.Background(), "Ym")
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if want := "Ym" + today() + "0001"; code != want {
		t.Fatalf("跨天后编号 = %q, 期望 %q", code, want)
	}
}

func TestSequenceIsMemoryOnly(t *testing.T) {
	// 新实例（等价进程重启）不继承历史计数，从 0001 开始
	seq := &Sequence{}
	for i := 0; i < 3; i++ {
		if _, err := seq.Next(context.Background(), "Ym"); err != nil {
			t.Fatalf("Next: %v", err)
		}
	}

	restarted := &Sequence{}
	code, err := restarted.Next(context.Background(), "Ym")
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if want := "Ym" + today() + "0001"; code != want {
		t.Fatalf("内存实现的计数不应跨实例保留，实际 %q", code)
	}
}

func TestSequenceConcurrentUnique(t *testing.T) {
	seq := &Sequence{}

	const n = 200
	codes := make([]string, n)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			code, err := seq.Next(context.Background(), "Ym")
			if err != nil {
				t.Errorf("Next: %v", err)
				return
			}
			codes[idx] = code
		}(i)
	}
	wg.Wait()

	seen := make(map[string]bool, n)
	for _, code := range codes {
		if seen[code] {
			t.Fatalf("编号重复: %s", code)
		}
		seen[code] = true
	}

	for want := 1; want <= n; want++ {
		if !seen[fmt.Sprintf("Ym%s%04d", today(), want)] {
			t.Fatalf("缺少序号 %d（共 %d 个）", want, len(seen))
		}
	}
}

func TestSequenceCtxCanceled(t *testing.T) {
	seq := &Sequence{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := seq.Next(ctx, "Ym"); err == nil {
		t.Fatal("ctx 已取消时应返回错误")
	}
}

// TestSequenceSingleton 校验 NewSequence 返回同一实例（§16.8：计数器是进程内唯一状态）。
func TestSequenceSingleton(t *testing.T) {
	first := NewSequence()
	second := NewSequence()

	if first != second {
		t.Fatal("NewSequence 应返回同一实例，否则多个计数器会各自计数导致重号")
	}
}
