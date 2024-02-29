package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/maliboot/mago/helper"
	"io"
	"os"
	"sync"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	_ "go.beyondstorage.io/services/cos/v3"
	_ "go.beyondstorage.io/services/oss/v3"
	"go.beyondstorage.io/v5/pairs"
	"go.beyondstorage.io/v5/services"
	"go.beyondstorage.io/v5/types"
	"golang.org/x/sync/errgroup"
)

type File struct {
	Default string `json:"default"`
	Local   struct {
		RootDir string `json:"root"`
	} `json:"local"`
	OSS struct {
		AccessId     string `json:"access_id"`
		AccessSecret string `json:"access_secret"`
		Bucket       string `json:"bucket"`
		Endpoint     string `json:"endpoint"`
	} `json:"oss"`
	COS struct {
		Region    string `json:"region"`
		AppId     string `json:"app_id"`
		SecretID  string `json:"secret_id"`
		SecretKey string `json:"secret_key"`
		Bucket    string `json:"bucket"`
	}
}

type FileStorage struct {
	name string
	ins  types.Storager
}

func (f *File) GetStorage(workDir string) (*FileStorage, error) {
	if f.Default == "" {
		return nil, fmt.Errorf("config:file未配置默认storage")
	}
	if workDir != "" && workDir[len(workDir)-1] != '/' {
		workDir += "/"
	}
	var connStr string
	switch f.Default {
	case "oss":
		connStr = fmt.Sprintf(
			"oss://%s%s?credential=hmac:%s:%s&endpoint=https:%s",
			f.OSS.Bucket,
			workDir,
			f.OSS.AccessId,
			f.OSS.AccessSecret,
			f.OSS.Endpoint,
		)
	default:
		return nil, fmt.Errorf("config:file配置的storage无效")
	}

	var fs FileStorage
	var err error
	fs.name = f.Default
	fs.ins, err = services.NewStoragerFromString(connStr)
	return &fs, err
}

func (s *FileStorage) Upload(path string, processFunc func(bs []byte)) error {
	return s.AutoUpload(path, processFunc, false)
}

func (s *FileStorage) AutoUpload(path string, processFunc func(bs []byte), auto bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// 获取文件状态信息
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	if auto && fileInfo.Size() > 10*1024*1024 {
		return s.MultipartUpload(path, func(part types.Part) {
			m, _ := helper.Marshal(part)
			processFunc(m)
		})
	}

	var opts = make([]types.Pair, 0)
	if processFunc != nil {
		opts = append(opts, pairs.WithIoCallback(processFunc))
	}
	_, err = s.ins.Write(fileInfo.Name(), file, fileInfo.Size(), opts...)
	if err != nil {
		return err
	}

	return nil
}

func (s *FileStorage) MultipartUpload(path string, processFunc func(part types.Part)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// 获取文件状态信息
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	multipartStorage, ok := s.ins.(types.Multiparter)
	if !ok {
		return fmt.Errorf("multiparter unimplemented for %s", s.name)
	}
	multipartIns, err := multipartStorage.CreateMultipart(fileInfo.Name())
	if err != nil {
		return fmt.Errorf("CreateMultipart %v: %v", path, err)
	}

	fileSize := fileInfo.Size()
	// 设置每个分片的大小,2MB
	partSize := int64(2 * 1024 * 1024)
	// 计算分片数量
	partNumber := int(fileSize/partSize) + 1

	// 逐个上传分片
	taskResult := make(chan *types.Part, 1)
	eg, egCtx := errgroup.WithContext(context.Background())
	var wg sync.WaitGroup
	for i := 0; i < partNumber; i++ {
		wg.Add(1)
		index := i
		eg.Go(func() error {
			defer wg.Done()
			// 读取分片数据
			partData := make([]byte, partSize)
			bytesRead, err := file.ReadAt(partData, partSize*int64(index))
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("WriteMultipart.Read %v: %v", path, err)
			}
			n, part, err := multipartStorage.WriteMultipart(multipartIns, bytes.NewReader(partData[:bytesRead]), int64(bytesRead), index)
			if err != nil {
				return fmt.Errorf("WriteMultipart %v: %v, len:%d", path, err, n)
			}
			taskResult <- part
			return nil
		})
	}
	go func() {
		// 等待上下文被取消
		<-egCtx.Done()
		// 取消所有协程的执行
		eg.Go(func() error {
			return egCtx.Err()
		})
	}()
	go func() {
		wg.Wait()
		close(taskResult) // 关闭结果通道
	}()

	var parts = make([]*types.Part, partNumber)
	for part := range taskResult {
		if processFunc != nil {
			go processFunc(*part)
		}
		parts[part.Index] = part
	}

	// 等待所有任务的结束，并返回第一个发生的错误
	if err = eg.Wait(); err != nil {
		return err
	}

	err = multipartStorage.CompleteMultipart(multipartIns, parts)
	if err != nil {
		return fmt.Errorf("CompleteMultipart %v: %v", path, err)
	}

	hlog.Debugf("path[%s] multipart upload size: %dMB", path, fileSize/1024/1024)
	return nil
}

func (s *FileStorage) DownloadFile(path string, targetPath string) error {
	_, err := s.ins.Stat(path)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var buf bytes.Buffer
	_, err = s.ins.Read(path, &buf)
	if err != nil {
		return err
	}

	_, err = file.Write(buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// MultiPartDownload 多块并行下载
// todo not final
func (s *FileStorage) MultiPartDownload(path string, targetFile *os.File, partOffset, partSize int64) error {

	var offset = partOffset
	var chunkSize int64 = 2 * 1024 * 1024
	var chunkNum = int(partSize/chunkSize) + 1

	for i := 0; i < chunkNum; i++ {
		index := i
		var buf bytes.Buffer
		if index+1 == chunkNum {
			chunkSize = partOffset + partSize - offset
		}
		bytesRead, err := s.ins.Read(path, &buf, pairs.WithOffset(offset), pairs.WithSize(chunkSize))
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if bytesRead == 0 {
			break
		}

		// 文件指定位置写入
		_, err = targetFile.WriteAt(buf.Bytes()[:bytesRead], offset)
		if err != nil {
			return err
		}
		offset += chunkSize
	}

	return nil
}
