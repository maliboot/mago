package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	_ "go.beyondstorage.io/services/cos/v3"
	_ "go.beyondstorage.io/services/fs/v4"
	_ "go.beyondstorage.io/services/oss/v3"
	"go.beyondstorage.io/v5/pairs"
	"go.beyondstorage.io/v5/services"
	"go.beyondstorage.io/v5/types"

	"github.com/maliboot/mago/helper"
)

type File struct {
	Default string `yaml:"default"`
	Local   struct {
		RootDir string `yaml:"root"`
	} `yaml:"local"`
	OSS struct {
		AccessId     string `yaml:"access_id"`
		AccessSecret string `yaml:"access_secret"`
		Bucket       string `yaml:"bucket"`
		Endpoint     string `yaml:"endpoint"`
	} `yaml:"oss"`
	COS struct {
		Region    string `yaml:"region"`
		AppId     string `yaml:"app_id"`
		SecretID  string `yaml:"secret_id"`
		SecretKey string `yaml:"secret_key"`
		Bucket    string `yaml:"bucket"`
	}
	Ctx any
}

type FileStorage struct {
	name            string
	ins             types.Storager
	uploadChunkSize int64
	bufPool         *helper.SyncBuf
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
	case "local":
		localAbs, err := filepath.Abs(f.Local.RootDir)
		if err != nil {
			return nil, fmt.Errorf("config:file配置的local异常:%v", err)
		}
		localAbs = filepath.Clean(localAbs)
		if err = os.MkdirAll(localAbs, 0755); err != nil {
			return nil, fmt.Errorf("config:file异常, 无法创建目录 %s: %w", localAbs, err)
		}
		if localAbs != "" && localAbs[len(localAbs)-1] != '/' {
			localAbs += "/"
		}
		connStr = fmt.Sprintf("fs://%s%s", localAbs, workDir)
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
	fs.uploadChunkSize = 1024 * 1024 * 2
	fs.bufPool = helper.NewSyncBuf()
	return &fs, err
}

func (f *File) IsFS() bool {
	return f.Default == "local"
}

func (f *File) GetFSAbsDir() (string, error) {
	localAbs, err := filepath.Abs(f.Local.RootDir)
	if err != nil {
		return "", fmt.Errorf("获取绝对路径失败: %v", err)
	}
	localAbs = filepath.Clean(localAbs)
	if err = os.MkdirAll(localAbs, 0755); err != nil {
		return "", fmt.Errorf("无法创建目录 %s: %w", localAbs, err)
	}

	return localAbs, nil
}

func (f *File) GetBaseUrl(path string) string {
	if path == "" {
		return ""
	}
	if path[0] != '/' {
		path = "/" + path
	}
	switch f.Default {
	case "local":
		return fmt.Sprintf("fs://%s", path)
	case "oss":
		return fmt.Sprintf("https://%s.%s%s", f.OSS.Bucket, f.OSS.Endpoint, path)
	}

	return ""
}

func (s *FileStorage) Stat(path string) (*types.Object, error) {
	return s.ins.Stat(path)
}

func (s *FileStorage) SetUploadChunkSize(chunkSize int64) {
	s.uploadChunkSize = chunkSize
}

func (s *FileStorage) isFS() bool {
	return s.name == "local"
}

func (s *FileStorage) upload(data interface{}, processFunc func(bs []byte), auto bool) error {
	var fileName string
	var fileSize int64
	var fileReader io.Reader
	var multipartFun func(fn func(part types.Part)) error = nil

	switch dataVal := data.(type) {
	case string:
		file, err := os.Open(dataVal)
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
		fileName = filepath.Base(file.Name())
		fileSize = fileInfo.Size()
		fileReader = file
		multipartFun = func(fn func(part types.Part)) error {
			return s.MultipartUploadByPath(dataVal, fn)
		}
	case *multipart.FileHeader:
		file, err := dataVal.Open()
		if err != nil {
			return err
		}
		defer func(file multipart.File) {
			_ = file.Close()
		}(file)

		fileName = filepath.Base(dataVal.Filename)
		fileSize = dataVal.Size
		fileReader = file
		multipartFun = func(fn func(part types.Part)) error {
			return s.MultipartUploadByFileHeader(dataVal, fn)
		}
	default:
		return errors.New("错误的data参数")
	}

	if auto && fileSize > 10*1024*1024 && !s.isFS() {
		var fn func(part types.Part) = nil
		if processFunc != nil {
			fn = func(part types.Part) {
				m, _ := helper.Marshal(part)
				processFunc(m)
			}
		}
		return multipartFun(fn)
	}

	var opts = make([]types.Pair, 0)
	if processFunc != nil {
		opts = append(opts, pairs.WithIoCallback(processFunc))
	}

	_, err := s.ins.Write(fileName, fileReader, fileSize, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (s *FileStorage) UploadByPath(path string, processFunc func(bs []byte), auto bool) error {
	return s.upload(path, processFunc, auto)
}

func (s *FileStorage) UploadByFileHeader(fileHeader *multipart.FileHeader, processFunc func(bs []byte), auto bool) error {
	return s.upload(fileHeader, processFunc, auto)
}

func (s *FileStorage) MultipartUploadByFileHeader(fileHeader *multipart.FileHeader, processFunc func(part types.Part)) error {
	f, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer func(f multipart.File) {
		_ = f.Close()
	}(f)

	return s.multipartUpload(fileHeader.Filename, fileHeader.Size, f, processFunc)
}

func (s *FileStorage) MultipartUploadByPath(path string, processFunc func(part types.Part)) error {
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

	return s.multipartUpload(fileInfo.Name(), fileInfo.Size(), file, processFunc)
}

func (s *FileStorage) multipartUpload(dstPath string, fileSize int64, srcReader io.Reader, processFunc func(part types.Part)) error {

	multipartStorage, ok := s.ins.(types.Multiparter)
	if !ok {
		return fmt.Errorf("multiparter unimplemented for %s", s.name)
	}
	multipartIns, err := multipartStorage.CreateMultipart(dstPath)
	if err != nil {
		return fmt.Errorf("CreateMultipart %v: %v", dstPath, err)
	}

	// 设置每个分片的大小,2MB
	partSize := s.uploadChunkSize
	offset := int64(0)
	index := 1
	var uploadParts []*types.Part
	for offset < fileSize {
		// 计算当前分片大小
		remaining := fileSize - offset
		currentPartSize := partSize
		if remaining < partSize {
			currentPartSize = remaining
		}

		n, part, err := multipartStorage.WriteMultipart(multipartIns, srcReader, currentPartSize, index)
		if err != nil {
			return fmt.Errorf("WriteMultipart %v: %v, len:%d", dstPath, err, n)
		}
		if processFunc != nil && part != nil {
			go processFunc(*part)
		}
		uploadParts = append(uploadParts, part)
		offset += currentPartSize
		index++
	}

	err = multipartStorage.CompleteMultipart(multipartIns, uploadParts)
	if err != nil {
		return fmt.Errorf("CompleteMultipart %v: %v", dstPath, err)
	}

	hlog.Debugf("path[%s] multipart upload size: %dMB", dstPath, fileSize/1024/1024)
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
