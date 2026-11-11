// Package filex 定义文件存储驱动抽象（local / 阿里云 OSS），并提供统一的 FileStorage 封装：
// 拼 workDir 前缀、按大小自动分片、追加写。config.File 只负责按配置装配驱动。
package filex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	// ErrNotExist 表示对象不存在。
	ErrNotExist = errors.New("filex: object not exist")
	// ErrInvalidKey 表示对象 key 非法（例如目录穿越）。
	ErrInvalidKey = errors.New("filex: invalid object key")
	// ErrMultipartUnsupported 表示当前驱动不支持分片上传。
	ErrMultipartUnsupported = errors.New("filex: multipart upload unsupported")
	// ErrAppendUnsupported 表示当前驱动不支持追加写。
	ErrAppendUnsupported = errors.New("filex: append unsupported")
	// ErrSignUnsupported 表示当前驱动不支持签名地址（如本地磁盘）。
	ErrSignUnsupported = errors.New("filex: signed url unsupported")
	// ErrExpiresInvalid 表示签名有效期非法。
	ErrExpiresInvalid = errors.New("filex: expires must be positive")
)

const (
	// defaultChunkSize 是分片上传与分段下载的默认块大小。
	defaultChunkSize int64 = 2 * 1024 * 1024
	// autoMultipartSize 超过该大小且驱动支持分片时自动走分片上传。
	autoMultipartSize int64 = 10 * 1024 * 1024
	// defaultSignExpires 是签名地址的默认有效期。
	defaultSignExpires = 15 * time.Minute
)

// Object 是存储对象的元信息。
type Object struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}

// SignedURL 是带签名的临时访问地址，过期后失效。私有读的桶靠它对外发链接。
type SignedURL struct {
	// URL 是可直接访问的地址。
	URL string
	// Method 是签名覆盖的 HTTP 方法。
	Method string
	// Expiration 是地址的失效时间。
	Expiration time.Time
}

// Part 是一个已完成的分片。
type Part struct {
	Number int
	Size   int64
	ETag   string
}

// Storage 是存储驱动必须实现的最小能力集。
type Storage interface {
	// Name 返回驱动名，如 local、oss。
	Name() string
	// Stat 返回对象元信息，不存在时返回 ErrNotExist。
	Stat(ctx context.Context, key string) (*Object, error)
	// Write 覆盖写入对象，size 小于 0 表示长度未知；onProgress 可为 nil。
	Write(ctx context.Context, key string, r io.Reader, size int64, onProgress func(written int64)) error
	// Append 追加写入对象，返回写入后的对象长度；对象不存在时按新建处理。
	Append(ctx context.Context, key string, r io.Reader, size int64, onProgress func(written int64)) (int64, error)
	// ReadRange 读取 [offset, offset+size)，size 小于等于 0 表示读到结尾。
	ReadRange(ctx context.Context, key string, offset, size int64) ([]byte, error)
	// ToFile 把对象下载到本地文件。
	ToFile(ctx context.Context, key, filePath string) error
	// Delete 删除对象，对象不存在时不视为错误。
	Delete(ctx context.Context, key string) error
	// URL 返回对象的固定访问地址（公有读桶可直接访问）。
	URL(key string) string
	// SignURL 生成带签名的临时访问地址，expires 为有效期，默认 15 分钟；
	// 不支持的驱动返回 ErrSignUnsupported。
	SignURL(ctx context.Context, key string, expires time.Duration) (*SignedURL, error)
}

// Multiparter 是驱动的可选能力：分片上传，并按分片回调。
type Multiparter interface {
	MultipartWrite(ctx context.Context, key string, r io.Reader, size, partSize int64, onPart func(Part)) error
}

// FileStorage 是对驱动的统一封装：方法接收不带前缀的业务路径，内部补 workDir。
type FileStorage struct {
	name            string
	ins             Storage
	workDir         string
	uploadChunkSize int64
}

// NewFileStorage 创建封装；workDir 会作为所有对象 key 的前缀。
func NewFileStorage(ins Storage, workDir string) *FileStorage {
	return &FileStorage{
		name:            ins.Name(),
		ins:             ins,
		workDir:         normalizeDir(workDir),
		uploadChunkSize: defaultChunkSize,
	}
}

// Name 返回底层驱动名。
func (s *FileStorage) Name() string { return s.name }

// Storage 暴露底层驱动：需要自定义 ctx 或驱动特有能力的调用方可直接使用。
func (s *FileStorage) Storage() Storage { return s.ins }

// Stat 返回对象元信息。
func (s *FileStorage) Stat(path string) (*Object, error) {
	return s.ins.Stat(context.Background(), s.key(path))
}

// URL 返回对象的访问地址。
func (s *FileStorage) URL(path string) string {
	return s.ins.URL(s.key(path))
}

// SignURL 生成临时访问地址，供私有读的桶对外发链接；expires 小于等于 0 时按 15 分钟处理。
func (s *FileStorage) SignURL(path string, expires time.Duration) (*SignedURL, error) {
	return s.ins.SignURL(context.Background(), s.key(path), expires)
}

// SetUploadChunkSize 设置分片大小，小于等于 0 时忽略。
func (s *FileStorage) SetUploadChunkSize(chunkSize int64) {
	if chunkSize > 0 {
		s.uploadChunkSize = chunkSize
	}
}

// IsFS 表示当前驱动是否为本地文件系统。
func (s *FileStorage) IsFS() bool {
	return s.name == "local"
}

// UploadByPath 上传本地文件；processFunc 逐次收到累计写入字节数（JSON 编码）。
func (s *FileStorage) UploadByPath(path string, processFunc func(bs []byte), auto bool) error {
	src, err := sourceFromPath(path)
	if err != nil {
		return err
	}
	defer func() { _ = src.close() }()

	return s.upload(src, processFunc, auto)
}

// UploadByFileHeader 上传表单文件；processFunc 逐次收到累计写入字节数。
func (s *FileStorage) UploadByFileHeader(fileHeader *multipart.FileHeader, processFunc func(bs []byte), auto bool) error {
	src, err := sourceFromHeader(fileHeader)
	if err != nil {
		return err
	}
	defer func() { _ = src.close() }()

	return s.upload(src, processFunc, auto)
}

// AppendByPath 追加写入本地文件内容；对象不存在时新建。
func (s *FileStorage) AppendByPath(path string, processFunc func(bs []byte)) error {
	src, err := sourceFromPath(path)
	if err != nil {
		return err
	}
	defer func() { _ = src.close() }()

	return s.append(src, processFunc)
}

// AppendByFileHeader 追加写入表单文件内容；对象不存在时新建。
func (s *FileStorage) AppendByFileHeader(fileHeader *multipart.FileHeader, processFunc func(bs []byte)) error {
	src, err := sourceFromHeader(fileHeader)
	if err != nil {
		return err
	}
	defer func() { _ = src.close() }()

	return s.append(src, processFunc)
}

// MultipartUploadByPath 强制分片上传本地文件。
func (s *FileStorage) MultipartUploadByPath(path string, processFunc func(part Part)) error {
	src, err := sourceFromPath(path)
	if err != nil {
		return err
	}
	defer func() { _ = src.close() }()

	return s.multipart(src, processFunc)
}

// MultipartUploadByFileHeader 强制分片上传表单文件。
func (s *FileStorage) MultipartUploadByFileHeader(fileHeader *multipart.FileHeader, processFunc func(part Part)) error {
	src, err := sourceFromHeader(fileHeader)
	if err != nil {
		return err
	}
	defer func() { _ = src.close() }()

	return s.multipart(src, processFunc)
}

// DownloadFile 下载对象到本地文件。
func (s *FileStorage) DownloadFile(path string, targetPath string) error {
	return s.ins.ToFile(context.Background(), s.key(path), targetPath)
}

// MultiPartDownload 用 range 读取把对象分段写入目标文件。
func (s *FileStorage) MultiPartDownload(path string, targetFile *os.File, partOffset, partSize int64) error {
	ctx := context.Background()
	key := s.key(path)

	if partSize <= 0 {
		obj, err := s.ins.Stat(ctx, key)
		if err != nil {
			return err
		}
		partSize = obj.Size - partOffset
	}

	for offset := partOffset; offset < partOffset+partSize; {
		current := defaultChunkSize
		if remain := partOffset + partSize - offset; remain < defaultChunkSize {
			current = remain
		}

		buf, err := s.ins.ReadRange(ctx, key, offset, current)
		if err != nil {
			return err
		}
		if len(buf) == 0 {
			return nil
		}

		if _, err = targetFile.WriteAt(buf, offset); err != nil {
			return err
		}

		offset += int64(len(buf))
	}

	return nil
}

// source 描述一次待上传的数据来源。
type source struct {
	name   string
	size   int64
	reader io.Reader
	closer io.Closer
}

func (s *source) close() error {
	if s.closer == nil {
		return nil
	}

	return s.closer.Close()
}

func sourceFromPath(path string) (*source, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	return &source{name: filepath.Base(path), size: info.Size(), reader: file, closer: file}, nil
}

func sourceFromHeader(fileHeader *multipart.FileHeader) (*source, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}

	return &source{name: filepath.Base(fileHeader.Filename), size: fileHeader.Size, reader: file, closer: file}, nil
}

// upload 按大小自动选择直传或分片上传。
func (s *FileStorage) upload(src *source, processFunc func(bs []byte), auto bool) error {
	if auto && !s.IsFS() && src.size > autoMultipartSize {
		if _, ok := s.ins.(Multiparter); ok {
			return s.multipart(src, func(part Part) { marshalTo(processFunc, part) })
		}
	}

	return s.ins.Write(context.Background(), s.key(src.name), src.reader, src.size, func(written int64) {
		marshalTo(processFunc, written)
	})
}

// append 追加写入，进度回调与 upload 保持同一套编码。
func (s *FileStorage) append(src *source, processFunc func(bs []byte)) error {
	_, err := s.ins.Append(context.Background(), s.key(src.name), src.reader, src.size, func(written int64) {
		marshalTo(processFunc, written)
	})

	return err
}

// multipart 调用驱动的分片能力；驱动不支持时退化为整段上传。
func (s *FileStorage) multipart(src *source, processFunc func(part Part)) error {
	multiparter, ok := s.ins.(Multiparter)
	if !ok {
		return s.ins.Write(context.Background(), s.key(src.name), src.reader, src.size, nil)
	}

	return multiparter.MultipartWrite(
		context.Background(), s.key(src.name), src.reader, src.size, s.uploadChunkSize, processFunc,
	)
}

// key 拼接 workDir 前缀并去掉前导斜杠。
func (s *FileStorage) key(path string) string {
	return s.workDir + strings.TrimPrefix(path, "/")
}

// normalizeDir 统一目录前缀格式，空串表示不加前缀。
func normalizeDir(workDir string) string {
	workDir = strings.TrimSpace(strings.Trim(workDir, "/"))
	if workDir == "" {
		return ""
	}

	return workDir + "/"
}

// marshalTo 把进度值 JSON 编码后交给回调；回调为 nil 时直接返回。
func marshalTo(processFunc func(bs []byte), value any) {
	if processFunc == nil {
		return
	}

	if bs, err := json.Marshal(value); err == nil {
		processFunc(bs)
	}
}

// progressReader 把读取进度回调出去，供调用方展示上传进度。
type progressReader struct {
	r    io.Reader
	read int64
	fn   func(written int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.read += int64(n)
		p.fn(p.read)
	}

	return n, err
}

// withProgress 在需要进度回调时包装 reader。
func withProgress(r io.Reader, fn func(written int64)) io.Reader {
	if fn == nil {
		return r
	}

	return &progressReader{r: r, fn: fn}
}
