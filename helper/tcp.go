package helper

import (
	"bytes"
	"encoding/hex"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// TcpLenPrefixedFramePack LV消息格式-打包
func TcpLenPrefixedFramePack(payload []byte) []byte {
	var buf bytes.Buffer
	_ = BufWriteUint32(&buf, uint32(len(payload)))
	buf.Write(payload)
	return buf.Bytes()
}

func TcpLenPrefixedFrameUnpack(payload []byte) ([]byte, uint32) {
	var buf = bytes.NewReader(payload)

	length, _ := BufReadUint32(buf)
	ret := make([]byte, length)
	_, err := buf.Read(ret)
	if err != nil {
		hlog.Error("TcpLenPrefixedFrameUnpack error: %s", hex.EncodeToString(payload))
		return nil, 0
	}
	return ret, length
}
