package mask

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

type TripleDes interface {
	EcbEncrypt(src []byte) (string, error)
	EcbDecrypt(src64 string) ([]byte, error)
	RandomSecretKey() (string, error)
}

type tripleDes struct {
	block cipher.Block
}

// New3Des 3des构造
func New3Des(secretKey string) (TripleDes, error) {
	decodeKey, err := base64.StdEncoding.DecodeString(secretKey)
	if err != nil {
		return nil, err
	}
	b, err := des.NewTripleDESCipher(decodeKey)
	if err != nil {
		return nil, err
	}

	return &tripleDes{block: b}, nil
}

// EcbEncrypt ecb模加密+PKCS5Padding
func (td tripleDes) EcbEncrypt(src []byte) (string, error) {
	bs := td.block.BlockSize()
	origData := PKCS5Padding(src, bs)
	if len(origData)%bs != 0 {
		return "", errors.New("need a multiple of the blocksize")
	}
	out := make([]byte, len(origData))
	dst := out
	for len(origData) > 0 {
		td.block.Encrypt(dst, origData[:bs])
		origData = origData[bs:]
		dst = dst[bs:]
	}

	outString := base64.StdEncoding.EncodeToString(out)
	return outString, nil
}

// EcbDecrypt ecb模解密+PKCS5UnPadding
func (td tripleDes) EcbDecrypt(src64 string) ([]byte, error) {
	src, err := base64.StdEncoding.DecodeString(src64)
	if err != nil {
		return []byte{}, err
	}

	bs := td.block.BlockSize()
	if len(src)%bs != 0 {
		return []byte{}, errors.New("crypto/cipher: input not full blocks")
	}
	out := make([]byte, len(src))
	dst := out
	for len(src) > 0 {
		td.block.Decrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	out = PKCS5UnPadding(out)
	return out, nil
}

// RandomSecretKey
// 随机一串3DES密钥，长度为24字节
func (td tripleDes) RandomSecretKey() (string, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
