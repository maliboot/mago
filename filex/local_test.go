package filex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalWriteStatReadDelete(t *testing.T) {
	dir := t.TempDir()
	local, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	ctx := context.Background()
	content := "hello filex"

	var written int64
	if err = local.Write(ctx, "a/b/note.txt", strings.NewReader(content), int64(len(content)), func(n int64) {
		written = n
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if written != int64(len(content)) {
		t.Fatalf("进度回调 = %d, 期望 %d", written, len(content))
	}

	obj, err := local.Stat(ctx, "a/b/note.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if obj.Size != int64(len(content)) {
		t.Fatalf("Size = %d, 期望 %d", obj.Size, len(content))
	}

	part, err := local.ReadRange(ctx, "a/b/note.txt", 6, 5)
	if err != nil {
		t.Fatalf("ReadRange: %v", err)
	}
	if string(part) != "filex" {
		t.Fatalf("ReadRange = %q, 期望 filex", part)
	}

	if err = local.Delete(ctx, "a/b/note.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = local.Stat(ctx, "a/b/note.txt"); !errors.Is(err, ErrNotExist) {
		t.Fatalf("删除后 Stat 错误 = %v, 期望 ErrNotExist", err)
	}
}

func TestLocalAppend(t *testing.T) {
	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	ctx := context.Background()

	// 首次追加等价于新建
	size, err := local.Append(ctx, "logs/app.log", strings.NewReader("line1\n"), 6, nil)
	if err != nil {
		t.Fatalf("Append 首次: %v", err)
	}
	if size != 6 {
		t.Fatalf("首次追加后长度 = %d, 期望 6", size)
	}

	var written int64
	size, err = local.Append(ctx, "logs/app.log", strings.NewReader("line2\n"), 6, func(n int64) { written = n })
	if err != nil {
		t.Fatalf("Append 第二次: %v", err)
	}
	if size != 12 {
		t.Fatalf("二次追加后长度 = %d, 期望 12", size)
	}
	if written != 6 {
		t.Fatalf("进度回调 = %d, 期望 6", written)
	}

	bs, err := local.ReadRange(ctx, "logs/app.log", 0, 0)
	if err != nil {
		t.Fatalf("ReadRange: %v", err)
	}
	if string(bs) != "line1\nline2\n" {
		t.Fatalf("内容 = %q", bs)
	}
}

func TestLocalToFile(t *testing.T) {
	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	ctx := context.Background()
	if err = local.Write(ctx, "src.txt", strings.NewReader("payload"), 7, nil); err != nil {
		t.Fatalf("Write: %v", err)
	}

	target := filepath.Join(t.TempDir(), "out", "dst.txt")
	if err = local.ToFile(ctx, "src.txt", target); err != nil {
		t.Fatalf("ToFile: %v", err)
	}

	bs, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(bs) != "payload" {
		t.Fatalf("内容 = %q, 期望 payload", bs)
	}
}

func TestLocalClampsTraversal(t *testing.T) {
	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	ctx := context.Background()
	if _, err = local.Stat(ctx, ""); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("空 key 错误 = %v, 期望 ErrInvalidKey", err)
	}

	// ".." 会被收敛到根目录内，而不是报错，更不能写到 root 之外。
	for _, key := range []string{"../escape.txt", "a/../../escape.txt"} {
		if err = local.Write(ctx, key, strings.NewReader("x"), 1, nil); err != nil {
			t.Fatalf("key %q Write = %v", key, err)
		}
	}

	if _, err = os.Stat(filepath.Join(local.Root(), "escape.txt")); err != nil {
		t.Fatalf("key 未收敛到 root 内: %v", err)
	}

	if _, err = os.Stat(filepath.Join(filepath.Dir(local.Root()), "escape.txt")); err == nil {
		t.Fatal("目录穿越未拦住：文件被写到 root 之外")
	}
}

func TestLocalMultipartUnsupported(t *testing.T) {
	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	if _, ok := any(local).(Multiparter); ok {
		t.Fatal("local 驱动不应实现 Multiparter")
	}
}

func TestLocalSignURLUnsupported(t *testing.T) {
	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	if _, err = local.SignURL(context.Background(), "a.txt", 0); !errors.Is(err, ErrSignUnsupported) {
		t.Fatalf("错误 = %v, 期望 ErrSignUnsupported", err)
	}
}

func TestLocalURL(t *testing.T) {
	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	if got := local.URL("a/b.txt"); !strings.HasPrefix(got, "file://"+local.Root()) {
		t.Fatalf("URL = %q, 期望以 file://%s 开头", got, local.Root())
	}
}
