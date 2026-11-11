package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maliboot/mago/filex"
)

// File 是文件存储配置，按 default 选择驱动（local / oss）；
// 读写能力见 filex.FileStorage。
type File struct {
	Default string `yaml:"default"`
	Local   struct {
		RootDir string `yaml:"root"`
	} `yaml:"local"`
	OSS struct {
		Region       string `yaml:"region"`
		Endpoint     string `yaml:"endpoint"`
		AccessID     string `yaml:"access_id"`
		AccessSecret string `yaml:"access_secret"`
		Bucket       string `yaml:"bucket"`
	} `yaml:"oss"`
	Ctx any
}

// GetStorage 按配置创建存储实例，workDir 会作为所有对象 key 的前缀。
func (f *File) GetStorage(workDir string) (*filex.FileStorage, error) {
	if f.Default == "" {
		return nil, errors.New("config:file 未配置 default")
	}

	ins, err := f.newStorage()
	if err != nil {
		return nil, err
	}

	return filex.NewFileStorage(ins, workDir), nil
}

// newStorage 按 default 装配驱动，未知驱动直接报错。
func (f *File) newStorage() (filex.Storage, error) {
	switch f.Default {
	case "local":
		ins, err := filex.NewLocal(f.Local.RootDir)
		if err != nil {
			return nil, fmt.Errorf("config:file local 配置异常: %w", err)
		}

		return ins, nil
	case "oss":
		ins, err := filex.NewOSS(filex.OSSOptions{
			Region:   f.OSS.Region,
			Endpoint: f.OSS.Endpoint,
			AccessID: f.OSS.AccessID,
			Secret:   f.OSS.AccessSecret,
			Bucket:   f.OSS.Bucket,
		})
		if err != nil {
			return nil, fmt.Errorf("config:file oss 配置异常: %w", err)
		}

		return ins, nil
	default:
		return nil, fmt.Errorf("config:file default 无效: %s", f.Default)
	}
}

// IsFS 表示当前使用本地文件系统存储。
func (f *File) IsFS() bool {
	return f.Default == "local"
}

// GetFSAbsDir 返回本地存储的绝对根目录，不存在时自动创建。
func (f *File) GetFSAbsDir() (string, error) {
	abs, err := filepath.Abs(f.Local.RootDir)
	if err != nil {
		return "", fmt.Errorf("获取绝对路径失败: %w", err)
	}

	abs = filepath.Clean(abs)
	if err = os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("无法创建目录 %s: %w", abs, err)
	}

	return abs, nil
}

// GetBaseUrl 返回对象的对外访问前缀 + path，local 用 file:// 协议。
func (f *File) GetBaseUrl(path string) string {
	if path == "" {
		return ""
	}

	if path[0] != '/' {
		path = "/" + path
	}

	switch f.Default {
	case "local":
		root, err := f.GetFSAbsDir()
		if err != nil {
			return ""
		}

		return "file://" + root + path
	case "oss":
		host := "oss-" + f.OSS.Region + ".aliyuncs.com"
		if f.OSS.Endpoint != "" {
			host = strings.TrimPrefix(strings.TrimPrefix(f.OSS.Endpoint, "https://"), "http://")
		}

		return fmt.Sprintf("https://%s.%s%s", f.OSS.Bucket, strings.TrimSuffix(host, "/"), path)
	}

	return ""
}
