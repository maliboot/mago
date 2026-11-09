package helper

import (
	"crypto/md5"
	"io"
	"os"
	"os/user"
	"runtime"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func FileMD5(file *os.File, chunkSize int64, chunkFn func(chunkBytes []byte)) ([]byte, error) {
	hash := md5.New()

	// 分块计算文件的 MD5
	if chunkSize == 0 {
		chunkSize = 1024 * 128 // 分块大小
	}
	buffer := make([]byte, chunkSize)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if chunkFn != nil {
			chunkFn(buffer[:n])
		}
		hash.Write(buffer[:n])
	}

	_, _ = file.Seek(0, io.SeekStart)
	// 计算 MD5 值并返回
	hashInBytes := hash.Sum(nil)
	return hashInBytes, nil
}

func SafeUserCacheDir() string {
	// 1. 尝试标准库
	if dir, err := os.UserCacheDir(); err == nil && dir != "" {
		return dir
	}

	// 2. 平台特定逻辑
	switch runtime.GOOS {
	case "windows":
		if dir := os.Getenv("LocalAppData"); dir != "" {
			return dir
		}
		if dir := os.Getenv("AppData"); dir != "" {
			return dir
		}
		if dir := os.Getenv("UserProfile"); dir != "" {
			return dir + `\AppData\Local`
		}
		return `C:\Windows\Temp`

	case "darwin": // macOS
		if home := os.Getenv("HOME"); home != "" {
			return home + "/Library/Caches"
		}
		if u, err := user.Current(); err == nil && u.HomeDir != "" {
			return u.HomeDir + "/Library/Caches"
		}
		return "/tmp"

	default: // Linux/Unix
		if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
			return dir
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/.cache"
		}
		if u, err := user.Current(); err == nil && u.HomeDir != "" {
			return u.HomeDir + "/.cache"
		}
		if os.Getuid() == 0 {
			return "/root/.cache"
		}
		return "/tmp/.cache"
	}
}

func BytesToMB(bytes uint64) float64 {
	return float64(bytes) / (1 << 20)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true // 文件存在
	}
	if os.IsNotExist(err) {
		return false // 文件不存在
	}
	// 其他错误（权限问题等），可根据需求处理
	hlog.Errorf("检查文件 %s 时出错: %v\n", path, err)
	return false
}

func FileRemove(p string) error {
	if !FileExists(p) {
		return nil
	}
	return os.Remove(p)
}
