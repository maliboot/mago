package filex

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

func TestNewOSSValidation(t *testing.T) {
	if _, err := NewOSS(OSSOptions{Region: "cn-hangzhou"}); err == nil {
		t.Fatal("缺少 bucket 时应报错")
	}

	if _, err := NewOSS(OSSOptions{Bucket: "demo"}); err == nil {
		t.Fatal("region 与 endpoint 都缺失时应报错")
	}

	if _, err := NewOSS(OSSOptions{Bucket: "demo", Region: "cn-hangzhou", AccessID: "ak", Secret: "sk"}); err != nil {
		t.Fatalf("region 配置齐全时不应报错: %v", err)
	}
}

func TestOSSURL(t *testing.T) {
	byRegion, err := NewOSS(OSSOptions{Bucket: "demo", Region: "cn-hangzhou"})
	if err != nil {
		t.Fatalf("NewOSS(region): %v", err)
	}
	if got, want := byRegion.URL("/a/b.png"), "https://demo.oss-cn-hangzhou.aliyuncs.com/a/b.png"; got != want {
		t.Fatalf("URL = %q, 期望 %q", got, want)
	}

	byEndpoint, err := NewOSS(OSSOptions{Bucket: "demo", Endpoint: "https://oss-cn-shanghai.aliyuncs.com/"})
	if err != nil {
		t.Fatalf("NewOSS(endpoint): %v", err)
	}
	if got, want := byEndpoint.URL("a/b.png"), "https://demo.oss-cn-shanghai.aliyuncs.com/a/b.png"; got != want {
		t.Fatalf("URL = %q, 期望 %q", got, want)
	}
	if !strings.HasPrefix(byEndpoint.Name(), "oss") {
		t.Fatalf("Name = %q", byEndpoint.Name())
	}
}

func TestDeriveRegion(t *testing.T) {
	cases := map[string]string{
		"oss-cn-hangzhou.aliyuncs.com":         "cn-hangzhou",
		"https://oss-cn-shanghai.aliyuncs.com": "cn-shanghai",
		"oss-cn-beijing-internal.aliyuncs.com": "cn-beijing",
		"oss-cn-shenzhen.aliyuncs.com/":        "cn-shenzhen",
		"":                                     "",
	}
	for endpoint, want := range cases {
		if got := deriveRegion(endpoint); got != want {
			t.Fatalf("deriveRegion(%q) = %q, 期望 %q", endpoint, got, want)
		}
	}
}

func TestOSSSignURL(t *testing.T) {
	o, err := NewOSS(OSSOptions{
		Bucket:   "demo-bucket",
		Region:   "cn-hangzhou",
		AccessID: "test-ak",
		Secret:   "test-sk",
	})
	if err != nil {
		t.Fatalf("NewOSS: %v", err)
	}

	before := time.Now()
	signed, err := o.SignURL(context.Background(), "private/a.png", 10*time.Minute)
	if err != nil {
		t.Fatalf("SignURL: %v", err)
	}

	if !strings.Contains(signed.URL, "demo-bucket.oss-cn-hangzhou.aliyuncs.com/private/a.png") {
		t.Fatalf("URL = %q, 期望包含桶域名与对象 key", signed.URL)
	}
	if signed.Method != "GET" {
		t.Fatalf("Method = %q, 期望 GET", signed.Method)
	}
	// 签名参数必须存在，且带上有效期
	for _, param := range []string{"x-oss-signature=", "x-oss-credential=", "x-oss-expires="} {
		if !strings.Contains(signed.URL, param) {
			t.Fatalf("URL 缺少签名参数 %s: %s", param, signed.URL)
		}
	}
	// 失效时间应落在 now+10m 附近
	if signed.Expiration.Before(before.Add(9*time.Minute)) || signed.Expiration.After(before.Add(11*time.Minute)) {
		t.Fatalf("Expiration = %v, 期望约 10 分钟后", signed.Expiration)
	}

	// 不传有效期时按默认 15 分钟
	def, err := o.SignURL(context.Background(), "private/a.png", 0)
	if err != nil {
		t.Fatalf("SignURL(默认有效期): %v", err)
	}
	if def.Expiration.Before(before.Add(14 * time.Minute)) {
		t.Fatalf("默认有效期 = %v, 期望约 15 分钟", def.Expiration)
	}
}

func TestWrapOSSErr(t *testing.T) {
	if err := wrapOSSErr(&oss.ServiceError{StatusCode: 404, Code: "NoSuchKey"}); !errors.Is(err, ErrNotExist) {
		t.Fatalf("404 应转成 ErrNotExist, 实际 %v", err)
	}

	origin := &oss.ServiceError{StatusCode: 403, Code: "AccessDenied"}
	if err := wrapOSSErr(origin); !errors.Is(err, origin) {
		t.Fatalf("非 404 错误应原样返回, 实际 %v", err)
	}
}
