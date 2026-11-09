package helper

import (
	"crypto/md5"
	"encoding/base64"
)

var (
	kAesLogKey = ""
)

func AesEncryptLog(stack []byte, key string) string {
	if key == "" && kAesLogKey != "" {
		key = kAesLogKey
	}
	if key == "" {
		return string(stack)
	}

	panicKey := md5.Sum([]byte(key))
	panicEncrypted, _ := AesCbcEncrypt(stack, panicKey[:])
	return base64.StdEncoding.EncodeToString(panicEncrypted)
}

func SetAesLogKey(key string) {
	kAesLogKey = key
}
