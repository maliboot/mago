package filex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newLocalStorage 构造一个 workDir 前缀为 uploads 的本地存储。
func newLocalStorage(t *testing.T, workDir string) (*FileStorage, *Local) {
	t.Helper()

	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	return NewFileStorage(local, workDir), local
}

func TestFileStorageKeyPrefix(t *testing.T) {
	fs, local := newLocalStorage(t, "/uploads/2026/")

	ctx := context.Background()
	if err := fs.Storage().Write(ctx, fs.key("a.txt"), strings.NewReader("x"), 1, nil); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// 业务路径不带前缀，落到驱动层时被补成 uploads/2026/a.txt
	if _, err := os.Stat(filepath.Join(local.Root(), "uploads/2026/a.txt")); err != nil {
		t.Fatalf("workDir 前缀未生效: %v", err)
	}

	obj, err := fs.Stat("/a.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if obj.Size != 1 {
		t.Fatalf("Size = %d, 期望 1", obj.Size)
	}
}

func TestFileStorageUploadAndAppend(t *testing.T) {
	fs, _ := newLocalStorage(t, "")

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "chunk.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var progress int64
	if err := fs.UploadByPath(src, func(bs []byte) { progress++ }, true); err != nil {
		t.Fatalf("UploadByPath: %v", err)
	}
	if progress == 0 {
		t.Fatal("上传进度回调未被触发")
	}

	if err := fs.AppendByPath(src, nil); err != nil {
		t.Fatalf("AppendByPath: %v", err)
	}
	if err := fs.AppendByPath(src, nil); err != nil {
		t.Fatalf("AppendByPath: %v", err)
	}

	bs, err := fs.Storage().ReadRange(context.Background(), fs.key("chunk.txt"), 0, 0)
	if err != nil {
		t.Fatalf("ReadRange: %v", err)
	}
	if want := "hellohellohello"; string(bs) != want {
		t.Fatalf("追加后内容 = %q, 期望 %q", bs, want)
	}
}

func TestFileStorageDownload(t *testing.T) {
	fs, _ := newLocalStorage(t, "docs")

	src := filepath.Join(t.TempDir(), "src.txt")
	if err := os.WriteFile(src, []byte("0123456789"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := fs.UploadByPath(src, nil, false); err != nil {
		t.Fatalf("UploadByPath: %v", err)
	}

	full := filepath.Join(t.TempDir(), "full.txt")
	if err := fs.DownloadFile("src.txt", full); err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	if bs, err := os.ReadFile(full); err != nil || string(bs) != "0123456789" {
		t.Fatalf("下载内容 = %q, %v", bs, err)
	}

	part := filepath.Join(t.TempDir(), "part.txt")
	target, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	defer func() { _ = target.Close() }()

	if err = fs.MultiPartDownload("src.txt", target, 3, 4); err != nil {
		t.Fatalf("MultiPartDownload: %v", err)
	}
	if bs, err := os.ReadFile(part); err != nil || string(bs[3:7]) != "3456" {
		t.Fatalf("分段下载内容 = %q, %v", bs, err)
	}
}
