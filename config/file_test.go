package config

import (
	"fmt"
	"go.beyondstorage.io/v5/types"
	"log"
	"testing"
)

func getOssFileIns() *File {
	ins := &File{}
	ins.Default = "oss"
	ins.OSS.AccessId = "xxx"
	ins.OSS.AccessSecret = "xxx"
	ins.OSS.Bucket = "xxx"
	ins.OSS.Endpoint = "oss-cn-beijing.aliyuncs.com"
	return ins
}

func TestFileStorage_Upload(t *testing.T) {
	type fields struct {
		name string
		ins  types.Storager
		err  error
	}
	type args struct {
		path        string
		processFunc func(bs []byte)
	}

	storage, err := getOssFileIns().GetStorage("/stone/test/")
	cur := int64(0)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "oss-upload",
			fields: fields{
				name: "oss",
				ins:  storage.ins,
				err:  err,
			},
			args: args{
				path: "/Users/stone/Downloads/test/stone.jpg",
				processFunc: func(bs []byte) {
					cur += int64(len(bs))
					log.Printf("write %d bytes already", cur)
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &FileStorage{
				name: tt.fields.name,
				ins:  tt.fields.ins,
			}
			if err := s.Upload(tt.args.path, tt.args.processFunc); (err != nil) != tt.wantErr {
				t.Errorf("Upload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileStorage_MultipartUpload(t *testing.T) {
	type fields struct {
		name string
		ins  types.Storager
	}
	type args struct {
		path        string
		processFunc func(part types.Part)
	}
	storage, _ := getOssFileIns().GetStorage("/stone/test2/")
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "file-multipartUpload",
			fields: fields{
				name: "oss",
				ins:  storage.ins,
			},
			args: args{
				path: "/Users/stone/Downloads/test/stone.jpg",
				processFunc: func(part types.Part) {
					fmt.Printf("第%d个文件切片[%s]，大小[%d]", part.Index, part.ETag, part.Size)
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &FileStorage{
				name: tt.fields.name,
				ins:  tt.fields.ins,
			}
			if err := s.MultipartUpload(tt.args.path, tt.args.processFunc); (err != nil) != tt.wantErr {
				t.Errorf("MultipartUpload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileStorage_DownloadFile(t *testing.T) {
	type fields struct {
		name string
		ins  types.Storager
	}
	type args struct {
		path       string
		targetPath string
	}
	storage, _ := getOssFileIns().GetStorage("/stone/test2/")
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "file-downloadFile",
			fields: fields{
				name: "oss",
				ins:  storage.ins,
			},
			args: args{
				path:       "stone.jpg",
				targetPath: "/Users/stone/Downloads/test/stone2.jpg",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &FileStorage{
				name: tt.fields.name,
				ins:  tt.fields.ins,
			}
			if err := s.DownloadFile(tt.args.path, tt.args.targetPath); (err != nil) != tt.wantErr {
				t.Errorf("DownloadFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
