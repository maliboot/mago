package helper

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

func AesGcmEncrypt(data, aesKey []byte) ([]byte, error) {
	if data == nil || len(data) == 0 {
		return make([]byte, 0), nil
	}
	iv := make([]byte, 12)
	if len(data) < 12 {
		_, _ = crand.Read(iv)
	} else {
		copy(iv, data[:12])
	}
	return AesGcmEncryptWithIv(data, aesKey, iv, nil)
}

func AesGcmDecrypt(data []byte, aesKey []byte) ([]byte, error) {
	if data == nil || len(data) == 0 {
		return make([]byte, 0), nil
	}
	iv := make([]byte, 12)
	copy(iv, data[:12])
	ciphertext := data[12:]
	return AesGcmDecryptWithIv(ciphertext, aesKey, iv, nil)
}

func AesGcmEncryptWithIv(data, aesKey, iv, additionalData []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to make block: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to make GCM: %v", err)
	}

	ciphertext := gcm.Seal(nil, iv, data, additionalData)
	return ciphertext, nil
}

func AesGcmDecryptWithIv(data []byte, aesKey []byte, iv, additionalData []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	out, err := gcm.Open(nil, iv, data, additionalData)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func AesCbcDecrypt(input, aesKey []byte) ([]byte, error) {
	if len(input) < 16 {
		return nil, fmt.Errorf("input len < 16")
	}
	iv := input[:16]
	encryptedData := input[16:]
	decrypt, err := AesCbcDecryptWithIv(encryptedData, aesKey, iv)
	if err != nil {
		return nil, err
	}
	return decrypt, nil
}

func AesCbcEncrypt(input, aesKey []byte) ([]byte, error) {
	newInput := make([]byte, len(input))
	copy(newInput, input)
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(crand.Reader, iv); err != nil {
		return nil, err
	}

	var encOut []byte
	if len(aesKey) != 0 {
		encrypt, err := AesCbcEncryptWithIv(newInput, aesKey, iv)
		if err != nil {
			return nil, err
		}
		encOut = append(iv, encrypt...)
	}

	return encOut, nil
}

func AesCbcEncryptWithIv(data, key, iv []byte) ([]byte, error) {
	// 创建AES块密码
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 创建CBC加密器
	encrypter := cipher.NewCBCEncrypter(block, iv)

	// pkcs7Pad 自动填充
	dataPadded := pkcs7Padding(data, aes.BlockSize)

	// 加密明文数据
	ciphertext := make([]byte, len(dataPadded))
	encrypter.CryptBlocks(ciphertext, dataPadded)

	return ciphertext, nil
}

func AesCbcDecryptWithIv(data, key, iv []byte) ([]byte, error) {
	// 创建AES块密码
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 创建CBC解密器
	encrypter := cipher.NewCBCDecrypter(block, iv)

	// 解密数据
	plaintext := make([]byte, len(data))
	encrypter.CryptBlocks(plaintext, data)

	// pkcs7unPad 去除填充
	plaintext = pkcs7UnPadding(plaintext)
	return plaintext, nil
}

// Padding方法 (PKCS7 Padding)
func pkcs7Padding(data []byte, blockSize int) []byte {
	paddingLength := blockSize - len(data)%blockSize
	padding := bytes.Repeat([]byte{byte(paddingLength)}, paddingLength)
	return append(data, padding...)
}

// Unpadding方法
func pkcs7UnPadding(data []byte) []byte {
	paddingLength := int(data[len(data)-1])
	return data[:len(data)-paddingLength]
}

func CreateEcdhKeyPair(curve ecdh.Curve) (*ecdh.PrivateKey, []byte, error) {
	if curve == nil {
		curve = ecdh.P256()
	}
	privateKey, err := ecdh.P256().GenerateKey(crand.Reader)
	if err != nil {
		return nil, nil, err
	}

	publickKey := privateKey.PublicKey()

	// 非压缩pubKeyBytes
	return privateKey, publickKey.Bytes(), nil
}

func GetEcdhPubKey(pubKeyBytes []byte, curve ecdh.Curve) (*ecdh.PublicKey, error) {
	if curve == nil {
		curve = ecdh.P256()
	}
	return curve.NewPublicKey(pubKeyBytes)
}

func GetEcdhSharedKey(myPriKey *ecdh.PrivateKey, otherPubKey *ecdh.PublicKey) ([]byte, error) {
	return myPriKey.ECDH(otherPubKey)
}

// CreateEcdsaKeyPair
// Deprecated: 交换密钥优先用ecdh，ecdsa主要用于验证
func CreateEcdsaKeyPair(curve elliptic.Curve) (*ecdsa.PrivateKey, []byte, error) {
	if curve == nil {
		curve = elliptic.P256()
	}
	privateKey, err := ecdsa.GenerateKey(curve, crand.Reader)
	if err != nil {
		return nil, nil, err
	}

	var compressedPubKey []byte
	// 标志位[Y坐标的奇偶性](1个字节) + x坐标值(32字节)
	compressedPubKey = elliptic.MarshalCompressed(curve, privateKey.PublicKey.X, privateKey.PublicKey.Y)
	return privateKey, compressedPubKey, nil
}

// Deprecated: 交换密钥优先用ecdh，ecdsa主要用于验证
func GetEcdsaPubKey(curve elliptic.Curve, compressedPubKey []byte) (*ecdsa.PublicKey, error) {
	if curve == nil {
		curve = elliptic.P256()
	}

	x, y := elliptic.UnmarshalCompressed(curve, compressedPubKey)
	if x == nil {
		return nil, errors.New("无效的椭圆点编码")
	}

	key := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}

	// Verify that the point is on the curve
	if !key.Curve.IsOnCurve(key.X, key.Y) {
		return nil, fmt.Errorf("point is not on the curve")
	}

	return key, nil
}

// Deprecated: 交换密钥优先用ecdh，ecdsa主要用于验证
func GetEcdsaSharedKey(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) ([]byte, error) {
	x, _ := publicKey.ScalarMult(publicKey.X, publicKey.Y, privateKey.D.Bytes())
	return x.Bytes(), nil
}

func AesGcmXorIV(iv []byte, index uint64) []byte {
	if len(iv) != 0xC {
		return []byte{}
	}

	newIv := make([]byte, len(iv))
	copy(newIv, iv)

	xorBlock := func(startPos, xorVal uint32) []byte {
		strVal := binary.BigEndian.Uint32(newIv[startPos : startPos+4])
		strVal ^= xorVal

		res := make([]byte, 4)
		binary.BigEndian.PutUint32(res, strVal)
		return res
	}
	hlLow32Bit := uint32(index)
	hlHigh32Bit := uint32(index >> 32)

	result := append(newIv[:0x4], xorBlock(0x4, hlHigh32Bit)...)
	result = append(result, xorBlock(0x8, hlLow32Bit)...)

	return result
}
