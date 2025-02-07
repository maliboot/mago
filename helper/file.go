package helper

import (
	"crypto/md5"
	"io"
	"os"
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
