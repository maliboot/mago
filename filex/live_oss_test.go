//go:build osslive

// 真实 OSS 联调测试：只在显式指定 tag 时运行。
//
//	cd filex && go test -tags osslive -run TestOSSLive -v -timeout 10m
//
// 凭据从仓库根的 runtime/oss.txt 读取（endpoint / bucket / accessKeyId / accessKeySecret），
// 代码里不落任何密钥。所有写入都发生在本次运行独有的前缀下，只删除本次创建的对象。
package filex_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maliboot/mago/filex"
)

const (
	// ossConfPath 是凭据文件相对包目录的位置。
	ossConfPath = "../runtime/oss.txt"
	// localSample 是联调用的样本文件。
	localSample = "../runtime/istone.MP4"
)

// ossConf 是 runtime/oss.txt 的解析结果。
type ossConf struct {
	Endpoint string
	Bucket   string
	KeyID    string
	Secret   string
}

// loadOSSConf 解析 key: value 形式的凭据文件。
func loadOSSConf(t *testing.T) ossConf {
	t.Helper()

	bs, err := os.ReadFile(ossConfPath)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", ossConfPath, err)
	}

	var conf ossConf
	for _, line := range strings.Split(string(bs), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}

		switch strings.TrimSpace(key) {
		case "endpoint":
			conf.Endpoint = strings.TrimSpace(value)
		case "bucket":
			conf.Bucket = strings.TrimSpace(value)
		case "accessKeyId":
			conf.KeyID = strings.TrimSpace(value)
		case "accessKeySecret":
			conf.Secret = strings.TrimSpace(value)
		}
	}

	if conf.Bucket == "" || conf.KeyID == "" || conf.Secret == "" {
		t.Fatalf("凭据文件缺少 bucket/accessKeyId/accessKeySecret")
	}

	return conf
}

// newLiveStorage 构造带独立前缀的 FileStorage。
func newLiveStorage(t *testing.T, workDir string) *filex.FileStorage {
	t.Helper()

	conf := loadOSSConf(t)
	driver, err := filex.NewOSS(filex.OSSOptions{
		Endpoint: conf.Endpoint,
		AccessID: conf.KeyID,
		Secret:   conf.Secret,
		Bucket:   conf.Bucket,
	})
	if err != nil {
		t.Fatalf("NewOSS: %v", err)
	}

	return filex.NewFileStorage(driver, workDir)
}

// sha256File 计算文件摘要，用于内容比对。
func sha256File(t *testing.T, path string) [32]byte {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("打开 %s 失败: %v", path, err)
	}
	defer func() { _ = file.Close() }()

	h := sha256.New()
	if _, err = io.Copy(h, file); err != nil {
		t.Fatalf("读取 %s 失败: %v", path, err)
	}

	var sum [32]byte
	copy(sum[:], h.Sum(nil))

	return sum
}

// fetchSigned 用签名地址真实下载并返回内容。
func fetchSigned(t *testing.T, signed *filex.SignedURL) []byte {
	t.Helper()

	resp, err := http.Get(signed.URL)
	if err != nil {
		t.Fatalf("GET 签名地址失败: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		t.Fatalf("签名地址状态码 = %d, body=%s", resp.StatusCode, body)
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取签名地址内容失败: %v", err)
	}

	return bs
}

func TestOSSLive(t *testing.T) {
	if _, err := os.Stat(localSample); err != nil {
		t.Fatalf("缺少样本文件 %s: %v", localSample, err)
	}

	workDir := fmt.Sprintf("_filex_live/%s", time.Now().Format("20060102T150405"))
	st := newLiveStorage(t, workDir)
	t.Logf("本次测试前缀: %s/", workDir)

	ctx := context.Background()
	sampleSize := fileSize(t, localSample)

	// 所有写入的对象都在这个前缀下，清理时只删这些 key。
	created := make([]string, 0, 8)
	cleanup := func() {
		for _, key := range created {
			if err := st.Storage().Delete(ctx, workDir+"/"+key); err != nil {
				t.Logf("清理 %s 失败: %v", key, err)
			}
		}
	}
	defer cleanup()

	t.Run("上传与元信息", func(t *testing.T) {
		var progress int64
		if err := st.UploadByPath(localSample, func(bs []byte) { progress++ }, false); err != nil {
			t.Fatalf("UploadByPath: %v", err)
		}
		created = append(created, filepath.Base(localSample))

		if progress == 0 {
			t.Fatal("进度回调未被触发")
		}

		obj, err := st.Stat(filepath.Base(localSample))
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if obj.Size != sampleSize {
			t.Fatalf("远端大小 = %d, 本地 = %d", obj.Size, sampleSize)
		}
		t.Logf("Stat: size=%d etag=%s type=%s", obj.Size, obj.ETag, obj.ContentType)
	})

	t.Run("签名地址可下载", func(t *testing.T) {
		signed, err := st.SignURL(filepath.Base(localSample), 10*time.Minute)
		if err != nil {
			t.Fatalf("SignURL: %v", err)
		}

		bs := fetchSigned(t, signed)
		if int64(len(bs)) != sampleSize {
			t.Fatalf("签名下载大小 = %d, 本地 = %d", len(bs), sampleSize)
		}
		if sha256.Sum256(bs) != sha256File(t, localSample) {
			t.Fatal("签名下载内容与本地文件不一致")
		}
		t.Logf("签名地址有效期至 %s", signed.Expiration.Format(time.RFC3339))
	})

	t.Run("Range 读取", func(t *testing.T) {
		key := workDir + "/" + filepath.Base(localSample)

		got, err := st.Storage().ReadRange(ctx, key, 1024, 2048)
		if err != nil {
			t.Fatalf("ReadRange: %v", err)
		}
		if len(got) != 2048 {
			t.Fatalf("Range 长度 = %d, 期望 2048", len(got))
		}

		local, err := readLocalRange(localSample, 1024, 2048)
		if err != nil {
			t.Fatalf("读取本地区间失败: %v", err)
		}
		if sha256.Sum256(got) != sha256.Sum256(local) {
			t.Fatal("Range 内容与本地不一致")
		}
	})

	t.Run("强制分片上传", func(t *testing.T) {
		st.SetUploadChunkSize(256 * 1024)

		var parts int
		if err := st.MultipartUploadByPath(localSample, func(part filex.Part) { parts++ }); err != nil {
			t.Fatalf("MultipartUploadByPath: %v", err)
		}

		// 分片上传按 base name 落对象，与上一步同名，覆盖写入
		obj, err := st.Stat(filepath.Base(localSample))
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if obj.Size != sampleSize {
			t.Fatalf("分片上传后大小 = %d, 期望 %d", obj.Size, sampleSize)
		}
		if parts < 2 {
			t.Fatalf("分片数 = %d, 期望 >= 2（分片大小 256KB）", parts)
		}

		// 合并后的内容必须与原文件逐字节一致
		merged := filepath.Join(t.TempDir(), "merged.MP4")
		if err = st.DownloadFile(filepath.Base(localSample), merged); err != nil {
			t.Fatalf("DownloadFile: %v", err)
		}
		if sha256File(t, merged) != sha256File(t, localSample) {
			t.Fatal("分片合并后内容与本地不一致")
		}
		t.Logf("分片数: %d，合并后内容校验通过", parts)
	})

	t.Run("自动模式大文件", func(t *testing.T) {
		bigPath := filepath.Join(t.TempDir(), "big.bin")
		if err := writeBigFile(bigPath, 12*1024*1024); err != nil {
			t.Fatalf("生成大文件失败: %v", err)
		}

		if err := st.UploadByPath(bigPath, nil, true); err != nil {
			t.Fatalf("UploadByPath(auto): %v", err)
		}
		created = append(created, "big.bin")

		obj, err := st.Stat("big.bin")
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if obj.Size != 12*1024*1024 {
			t.Fatalf("大小 = %d, 期望 12MB", obj.Size)
		}
	})

	t.Run("追加写", func(t *testing.T) {
		part1 := filepath.Join(t.TempDir(), "part1.txt")
		part2 := filepath.Join(t.TempDir(), "part2.txt")
		if err := os.WriteFile(part1, []byte("hello "), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if err := os.WriteFile(part2, []byte("oss"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		// 用同一个文件名，第一次 Append 创建 Appendable 对象
		if err := st.AppendByPath(part1, nil); err != nil {
			t.Fatalf("AppendByPath 首次: %v", err)
		}
		appendKey := filepath.Base(part1)
		created = append(created, appendKey)

		// 第二次追加：换成 part2 的内容，base name 不同，所以先复制成同名文件再追加
		sameName := filepath.Join(t.TempDir(), appendKey)
		if err := os.WriteFile(sameName, []byte("oss"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if err := st.AppendByPath(sameName, nil); err != nil {
			t.Fatalf("AppendByPath 二次: %v", err)
		}

		bs, err := st.Storage().ReadRange(ctx, workDir+"/"+appendKey, 0, 0)
		if err != nil {
			t.Fatalf("ReadRange: %v", err)
		}
		if string(bs) != "hello oss" {
			t.Fatalf("追加后内容 = %q, 期望 %q", bs, "hello oss")
		}
	})

	t.Run("非追加类型拒绝追加", func(t *testing.T) {
		// 由 PutObject 创建的对象不是 Appendable，OSS 应拒绝 Append
		if err := st.AppendByPath(localSample, nil); err == nil {
			t.Fatal("对 PutObject 创建的对象追加应失败")
		} else {
			t.Logf("预期内的拒绝: %v", err)
		}
	})

	t.Run("下载到本地与分段下载", func(t *testing.T) {
		full := filepath.Join(t.TempDir(), "full.MP4")
		if err := st.DownloadFile(filepath.Base(localSample), full); err != nil {
			t.Fatalf("DownloadFile: %v", err)
		}
		if sha256File(t, full) != sha256File(t, localSample) {
			t.Fatal("下载内容与本地不一致")
		}

		target := filepath.Join(t.TempDir(), "part.MP4")
		file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatalf("OpenFile: %v", err)
		}
		defer func() { _ = file.Close() }()

		if err = st.MultiPartDownload(filepath.Base(localSample), file, 1000, 3000); err != nil {
			t.Fatalf("MultiPartDownload: %v", err)
		}
		if err = file.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}

		got, err := readLocalRange(target, 1000, 3000)
		if err != nil {
			t.Fatalf("读取分段结果失败: %v", err)
		}
		want, err := readLocalRange(localSample, 1000, 3000)
		if err != nil {
			t.Fatalf("读取样本区间失败: %v", err)
		}
		if sha256.Sum256(got) != sha256.Sum256(want) {
			t.Fatal("分段下载内容与本地不一致")
		}
	})

	t.Run("删除本次创建的对象", func(t *testing.T) {
		count := len(created)
		for _, key := range created {
			if err := st.Storage().Delete(ctx, workDir+"/"+key); err != nil {
				t.Fatalf("Delete %s: %v", key, err)
			}
			if _, err := st.Storage().Stat(ctx, workDir+"/"+key); err == nil {
				t.Fatalf("%s 删除后仍可 Stat", key)
			} else if !strings.Contains(err.Error(), "not exist") {
				t.Fatalf("%s 删除后错误 = %v, 期望不存在", key, err)
			}
		}
		created = nil
		t.Logf("已删除本次写入的 %d 个对象", count)
	})
}

// fileSize 返回文件字节数。
func fileSize(t *testing.T, path string) int64 {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat %s: %v", path, err)
	}

	return info.Size()
}

// readLocalRange 读取文件指定区间。
func readLocalRange(path string, offset, size int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	buf := make([]byte, size)
	if _, err = file.ReadAt(buf, offset); err != nil && err != io.EOF {
		return nil, err
	}

	return buf, nil
}

// writeBigFile 生成指定大小、内容可校验的文件。
func writeBigFile(path string, size int64) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	buf := make([]byte, 1024*1024)
	for i := range buf {
		buf[i] = byte(i % 251)
	}

	for written := int64(0); written < size; {
		n := int64(len(buf))
		if remain := size - written; remain < n {
			n = remain
		}
		if _, err = file.Write(buf[:n]); err != nil {
			return err
		}
		written += n
	}

	return file.Sync()
}
