package filex

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Local 是基于本地文件系统的驱动。
type Local struct {
	root string
}

// NewLocal 创建本地驱动：root 转为绝对路径，目录不存在时自动创建。
func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	abs = filepath.Clean(abs)
	if err = os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}

	return &Local{root: abs}, nil
}

func (l *Local) Name() string { return "local" }

// Root 返回本地存储的绝对根目录。
func (l *Local) Root() string { return l.root }

func (l *Local) URL(key string) string {
	p, err := l.resolve(key)
	if err != nil {
		return ""
	}

	return "file://" + p
}

// SignURL 本地磁盘没有签名机制，直接返回 ErrSignUnsupported；本地开发请用 URL。
func (l *Local) SignURL(_ context.Context, _ string, _ time.Duration) (*SignedURL, error) {
	return nil, ErrSignUnsupported
}

func (l *Local) Stat(_ context.Context, key string) (*Object, error) {
	p, err := l.resolve(key)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(p)
	if err != nil {
		return nil, wrapErr(err)
	}

	return &Object{Key: key, Size: info.Size(), LastModified: info.ModTime()}, nil
}

func (l *Local) Write(_ context.Context, key string, r io.Reader, _ int64, onProgress func(written int64)) error {
	p, err := l.resolve(key)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, withProgress(r, onProgress))

	return err
}

func (l *Local) Append(_ context.Context, key string, r io.Reader, _ int64, onProgress func(written int64)) (int64, error) {
	p, err := l.resolve(key)
	if err != nil {
		return 0, err
	}

	if err = os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return 0, err
	}

	file, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}

	if _, err = io.Copy(file, withProgress(r, onProgress)); err != nil {
		_ = file.Close()
		return 0, err
	}

	if err = file.Close(); err != nil {
		return 0, err
	}

	info, err := os.Stat(p)
	if err != nil {
		return 0, wrapErr(err)
	}

	return info.Size(), nil
}

func (l *Local) ReadRange(_ context.Context, key string, offset, size int64) ([]byte, error) {
	p, err := l.resolve(key)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(p)
	if err != nil {
		return nil, wrapErr(err)
	}
	defer func() { _ = file.Close() }()

	if offset > 0 {
		if _, err = file.Seek(offset, io.SeekStart); err != nil {
			return nil, err
		}
	}

	var reader io.Reader = file
	if size > 0 {
		reader = io.LimitReader(file, size)
	}

	return io.ReadAll(reader)
}

func (l *Local) ToFile(_ context.Context, key, filePath string) error {
	src, err := l.resolve(key)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	return copyFile(src, filePath)
}

func (l *Local) Delete(_ context.Context, key string) error {
	p, err := l.resolve(key)
	if err != nil {
		return err
	}

	if err = os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

// resolve 把对象 key 映射到本地绝对路径，并阻止目录穿越。
func (l *Local) resolve(key string) (string, error) {
	clean := filepath.Clean(string(filepath.Separator) + key)
	if clean == string(filepath.Separator) {
		return "", ErrInvalidKey
	}

	full := l.root + clean
	if !strings.HasPrefix(full, l.root+string(filepath.Separator)) {
		return "", ErrInvalidKey
	}

	return full, nil
}

// copyFile 复制文件内容到目标路径。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return wrapErr(err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}

	if _, err = io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}

	return out.Close()
}

// wrapErr 把文件不存在的错误统一成 ErrNotExist。
func wrapErr(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotExist
	}

	return err
}
