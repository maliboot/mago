package helper

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"unsafe"
)

func GetArchBit() int {
	return int(unsafe.Sizeof(0))
}

func BufWriteUint(buf io.Writer, data uint) error {
	if GetArchBit() == 8 {
		return binary.Write(buf, binary.BigEndian, uint64(data))
	}
	return binary.Write(buf, binary.BigEndian, uint32(data))
}

func BufWriteUShort(buf io.Writer, data uint16) error {
	return binary.Write(buf, binary.BigEndian, data)
}

func BufWriteUint32(buf io.Writer, data uint32) error {
	return binary.Write(buf, binary.BigEndian, data)
}

func BufWriteUint64(buf io.Writer, data uint64) error {
	return binary.Write(buf, binary.BigEndian, data)
}

func BufReadUint(buf io.Reader) (uint, error) {
	var err error
	if GetArchBit() == 8 {
		var data64 uint64
		err = binary.Read(buf, binary.BigEndian, &data64)
		return uint(data64), err
	}

	var data32 uint32
	err = binary.Read(buf, binary.BigEndian, &data32)
	return uint(data32), err
}

func BufReadUShort(buf io.Reader) (uint16, error) {
	var data uint16
	var err = binary.Read(buf, binary.BigEndian, &data)
	return data, err
}

func BufReadUint32(buf io.Reader) (uint32, error) {
	var data uint32
	var err = binary.Read(buf, binary.BigEndian, &data)
	return data, err
}

func BufReadUint64(buf io.Reader) (uint64, error) {
	var data uint64
	var err = binary.Read(buf, binary.BigEndian, &data)
	return data, err
}

func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	zipWriter, err := zlib.NewWriterLevel(&buf, zlib.DefaultCompression)
	if err != nil {
		return nil, err
	}

	_, err = zipWriter.Write(data)
	if err != nil {
		return nil, err
	}
	_ = zipWriter.Close()

	return buf.Bytes(), nil
}

func Decompress(compressed []byte) ([]byte, error) {
	var decompress bytes.Buffer

	// 创建 zlib reader
	zlibReader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer func(zlibReader io.ReadCloser) {
		_ = zlibReader.Close()
	}(zlibReader)

	// 将解压缩的数据写入 buffer
	_, err = io.Copy(&decompress, zlibReader)
	if err != nil {
		return nil, err
	}

	return decompress.Bytes(), nil
}

func BytesToStr(data []byte) string {
	if data == nil || len(data) == 0 {
		return ""
	}

	return string(data)
}
