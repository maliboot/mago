package mask

import (
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
)

type RsaAuth interface {
	Sign(origData []byte) (string, error)
	Verify(signData string, origData []byte) error
}

type rsaAuth struct {
	innerPriKey []byte
	outerPubKey []byte
}

func NewRsaAuth(iPriKey, oPubKey string) RsaAuth {
	return &rsaAuth{
		innerPriKey: getPriKeyPem(iPriKey),
		outerPubKey: getPubKeyPem(oPubKey),
	}
}

func (a rsaAuth) Sign(origData []byte) (string, error) {
	m5 := md5.Sum(origData)
	hashed := md5.Sum([]byte(fmt.Sprintf("%X", m5)))
	b, _ := pem.Decode(a.innerPriKey)
	if b == nil {
		return "", errors.New("private key error")
	}
	privateKey, err := x509.ParsePKCS8PrivateKey(b.Bytes)
	if err != nil {
		return "", err
	}
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey.(*rsa.PrivateKey), crypto.MD5, hashed[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (a rsaAuth) Verify(signData string, origData []byte) error {
	sign, err := base64.StdEncoding.DecodeString(signData)
	if err != nil {
		return err
	}
	b, _ := pem.Decode(a.outerPubKey)
	if b == nil {
		return errors.New("pub key error")
	}
	pub, err := x509.ParsePKIXPublicKey(b.Bytes)
	if err != nil {
		return err
	}
	m5 := md5.Sum(origData)
	hashed := md5.Sum([]byte(fmt.Sprintf("%X", m5)))
	return rsa.VerifyPKCS1v15(pub.(*rsa.PublicKey), crypto.MD5, hashed[:], sign)
}

func getPriKeyPem(priKeyStr string) []byte {
	pKey := chunkSplit(priKeyStr, 64, "\n")
	pKeyStr := fmt.Sprintf("-----BEGIN PRIVATE KEY-----\n%s-----END PRIVATE KEY-----", pKey)
	return []byte(pKeyStr)
}

func getPubKeyPem(pubKeyStr string) []byte {
	pKey := chunkSplit(pubKeyStr, 64, "\n")
	pKeyStr := fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s-----END PUBLIC KEY-----", pKey)
	return []byte(pKeyStr)
}

func chunkSplit(body string, chunklen uint, end string) string {
	if end == "" {
		end = "\r\n"
	}
	runes, erunes := []rune(body), []rune(end)
	l := uint(len(runes))
	if l <= 1 || l < chunklen {
		return body + end
	}
	ns := make([]rune, 0, len(runes)+len(erunes))
	var i uint
	for i = 0; i < l; i += chunklen {
		if i+chunklen > l {
			ns = append(ns, runes[i:]...)
		} else {
			ns = append(ns, runes[i:i+chunklen]...)
		}
		ns = append(ns, erunes...)
	}
	return string(ns)
}
