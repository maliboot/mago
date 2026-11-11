package filex

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// OSSOptions 是阿里云 OSS 驱动的配置。
type OSSOptions struct {
	// Region 形如 cn-hangzhou；未配置 Endpoint 时必填。
	Region string
	// Endpoint 形如 oss-cn-hangzhou.aliyuncs.com，可带 https:// 前缀；配置后优先用于生成访问域名。
	Endpoint string
	AccessID string
	Secret   string
	Bucket   string
}

// OSS 是基于阿里云官方 SDK v2（alibabacloud-oss-go-sdk-v2）的驱动。
type OSS struct {
	client  *oss.Client
	bucket  string
	baseURL string
}

// NewOSS 创建 OSS 驱动：region 与 endpoint 至少配置一个。
func NewOSS(opt OSSOptions) (*OSS, error) {
	if opt.Bucket == "" {
		return nil, errors.New("filex: oss 未配置 bucket")
	}
	if opt.Region == "" && opt.Endpoint == "" {
		return nil, errors.New("filex: oss 至少需要配置 region 或 endpoint")
	}

	region := opt.Region
	if region == "" {
		region = deriveRegion(opt.Endpoint)
	}
	if region == "" {
		return nil, errors.New("filex: oss 无法确定 region，请显式配置 region")
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(opt.AccessID, opt.Secret)).
		WithRegion(region)

	host := "oss-" + region + ".aliyuncs.com"
	if opt.Endpoint != "" {
		endpoint := opt.Endpoint
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "https://" + endpoint
		}
		cfg = cfg.WithEndpoint(endpoint)
		host = trimScheme(endpoint)
	}

	return &OSS{
		client:  oss.NewClient(cfg),
		bucket:  opt.Bucket,
		baseURL: "https://" + opt.Bucket + "." + host,
	}, nil
}

func (o *OSS) Name() string { return "oss" }

func (o *OSS) URL(key string) string {
	return o.baseURL + "/" + strings.TrimPrefix(key, "/")
}

// SignURL 用官方 Presign 生成 GET 临时地址，纯本地计算签名，不发请求。
func (o *OSS) SignURL(ctx context.Context, key string, expires time.Duration) (*SignedURL, error) {
	if expires <= 0 {
		expires = defaultSignExpires
	}

	out, err := o.client.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(o.bucket),
		Key:    oss.Ptr(key),
	}, oss.PresignExpires(expires))
	if err != nil {
		return nil, wrapOSSErr(err)
	}

	return &SignedURL{URL: out.URL, Method: out.Method, Expiration: out.Expiration}, nil
}

func (o *OSS) Stat(ctx context.Context, key string) (*Object, error) {
	out, err := o.client.HeadObject(ctx, &oss.HeadObjectRequest{
		Bucket: oss.Ptr(o.bucket),
		Key:    oss.Ptr(key),
	})
	if err != nil {
		return nil, wrapOSSErr(err)
	}

	obj := &Object{Key: key, Size: out.ContentLength}
	if out.ETag != nil {
		obj.ETag = strings.Trim(*out.ETag, `"`)
	}
	if out.ContentType != nil {
		obj.ContentType = *out.ContentType
	}
	if out.LastModified != nil {
		obj.LastModified = *out.LastModified
	}

	return obj, nil
}

func (o *OSS) Write(ctx context.Context, key string, r io.Reader, size int64, onProgress func(written int64)) error {
	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(o.bucket),
		Key:    oss.Ptr(key),
		Body:   r,
	}
	if size >= 0 {
		req.ContentLength = oss.Ptr(size)
	}
	if onProgress != nil {
		req.ProgressFn = func(_, transferred, _ int64) { onProgress(transferred) }
	}

	_, err := o.client.PutObject(ctx, req)

	return wrapOSSErr(err)
}

// Append 追加写入；对象不存在时以 position=0 新建。
// 注意：OSS 只允许对 Appendable 类型的对象追加（即此前用 AppendObject 创建的对象），
// 对 PutObject 创建的对象追加会被 OSS 拒绝，错误原样返回。
func (o *OSS) Append(ctx context.Context, key string, r io.Reader, size int64, onProgress func(written int64)) (int64, error) {
	position := int64(0)
	if obj, err := o.Stat(ctx, key); err == nil {
		position = obj.Size
	} else if !errors.Is(err, ErrNotExist) {
		return 0, err
	}

	req := &oss.AppendObjectRequest{
		Bucket:   oss.Ptr(o.bucket),
		Key:      oss.Ptr(key),
		Position: oss.Ptr(position),
		Body:     r,
	}
	if size >= 0 {
		req.ContentLength = oss.Ptr(size)
	}
	if onProgress != nil {
		req.ProgressFn = func(_, transferred, _ int64) { onProgress(transferred) }
	}

	out, err := o.client.AppendObject(ctx, req)
	if err != nil {
		return 0, wrapOSSErr(err)
	}

	return out.NextPosition, nil
}

func (o *OSS) ReadRange(ctx context.Context, key string, offset, size int64) ([]byte, error) {
	req := &oss.GetObjectRequest{Bucket: oss.Ptr(o.bucket), Key: oss.Ptr(key)}
	switch {
	case size > 0:
		req.Range = oss.Ptr(fmt.Sprintf("bytes=%d-%d", offset, offset+size-1))
	case offset > 0:
		req.Range = oss.Ptr(fmt.Sprintf("bytes=%d-", offset))
	}

	out, err := o.client.GetObject(ctx, req)
	if err != nil {
		return nil, wrapOSSErr(err)
	}
	defer func() { _ = out.Body.Close() }()

	return io.ReadAll(out.Body)
}

func (o *OSS) ToFile(ctx context.Context, key, filePath string) error {
	_, err := o.client.GetObjectToFile(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(o.bucket),
		Key:    oss.Ptr(key),
	}, filePath)

	return wrapOSSErr(err)
}

func (o *OSS) Delete(ctx context.Context, key string) error {
	_, err := o.client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(o.bucket),
		Key:    oss.Ptr(key),
	})

	return wrapOSSErr(err)
}

// MultipartWrite 走 OSS 官方分片上传：初始化 → 逐片上传 → 完成，中途失败自动中止。
func (o *OSS) MultipartWrite(ctx context.Context, key string, r io.Reader, size, partSize int64, onPart func(Part)) error {
	if partSize <= 0 {
		partSize = 2 * 1024 * 1024
	}

	init, err := o.client.InitiateMultipartUpload(ctx, &oss.InitiateMultipartUploadRequest{
		Bucket: oss.Ptr(o.bucket),
		Key:    oss.Ptr(key),
	})
	if err != nil {
		return wrapOSSErr(err)
	}

	uploadID := init.UploadId
	done := false
	defer func() {
		if !done {
			_, _ = o.client.AbortMultipartUpload(ctx, &oss.AbortMultipartUploadRequest{
				Bucket:   oss.Ptr(o.bucket),
				Key:      oss.Ptr(key),
				UploadId: uploadID,
			})
		}
	}()

	parts := make([]oss.UploadPart, 0, size/partSize+1)
	for offset, number := int64(0), int32(1); offset < size; number++ {
		current := partSize
		if remain := size - offset; remain < partSize {
			current = remain
		}

		// 每片先读进内存再提交：body 长度与 ContentLength 严格一致，
		// 且 Body 可重复读，SDK 内部重试不会拿到被消费过的流。单片内存占用等于分片大小。
		buf := make([]byte, current)
		if _, err := io.ReadFull(r, buf); err != nil {
			return wrapOSSErr(err)
		}

		part, err := o.client.UploadPart(ctx, &oss.UploadPartRequest{
			Bucket:        oss.Ptr(o.bucket),
			Key:           oss.Ptr(key),
			UploadId:      uploadID,
			PartNumber:    number,
			Body:          bytes.NewReader(buf),
			ContentLength: oss.Ptr(current),
		})
		if err != nil {
			return wrapOSSErr(err)
		}

		parts = append(parts, oss.UploadPart{PartNumber: number, ETag: part.ETag})
		if onPart != nil {
			etag := ""
			if part.ETag != nil {
				etag = strings.Trim(*part.ETag, `"`)
			}
			onPart(Part{Number: int(number), Size: current, ETag: etag})
		}

		offset += current
	}

	_, err = o.client.CompleteMultipartUpload(ctx, &oss.CompleteMultipartUploadRequest{
		Bucket:                  oss.Ptr(o.bucket),
		Key:                     oss.Ptr(key),
		UploadId:                uploadID,
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{Parts: parts},
	})
	if err != nil {
		return wrapOSSErr(err)
	}

	done = true

	return nil
}

// wrapOSSErr 把 OSS 的 404 统一成 ErrNotExist，其余错误原样返回。
func wrapOSSErr(err error) error {
	if err == nil {
		return nil
	}

	var serr *oss.ServiceError
	if errors.As(err, &serr) && (serr.StatusCode == http.StatusNotFound || serr.Code == "NoSuchKey") {
		return ErrNotExist
	}

	return err
}

// deriveRegion 从 endpoint 推导 region，如 oss-cn-hangzhou.aliyuncs.com → cn-hangzhou。
func deriveRegion(endpoint string) string {
	host := trimScheme(endpoint)
	if host == "" {
		return ""
	}
	if i := strings.Index(host, "."); i > 0 {
		host = host[:i]
	}

	host = strings.TrimPrefix(host, "oss-")
	host = strings.TrimSuffix(host, "-internal")
	host = strings.TrimSuffix(host, "-accelerate")

	// 形如 cn-hangzhou / ap-southeast-1 的才算 region，其余交回调用方显式配置。
	if strings.Count(host, "-") < 1 || !strings.ContainsAny(host, "abcdefghijklmnopqrstuvwxyz") {
		return ""
	}

	return host
}

// trimScheme 去掉协议前缀与末尾斜杠，得到纯域名。
func trimScheme(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	return strings.TrimSuffix(endpoint, "/")
}
