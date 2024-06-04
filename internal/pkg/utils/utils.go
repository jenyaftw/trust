package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"math"
	"math/big"
)

func GetTLSConfig(certEnc string, keyEnc string, caEnc *string) (*tls.Config, error) {
	certString, err := base64.StdEncoding.DecodeString(certEnc)
	if err != nil {
		return nil, err
	}

	keyString, err := base64.StdEncoding.DecodeString(keyEnc)
	if err != nil {
		return nil, err
	}

	cer, err := tls.X509KeyPair(certString, keyString)
	if err != nil {
		return nil, err
	}

	config := tls.Config{
		Certificates:       []tls.Certificate{cer},
		InsecureSkipVerify: caEnc == nil,
	}

	if caEnc != nil {
		caString, err := base64.StdEncoding.DecodeString(*caEnc)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caString)

		config.RootCAs = caCertPool
		config.ClientCAs = caCertPool
		config.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return &config, nil
}

func GenerateRandomId() uint64 {
	rint, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	return rint.Uint64()
}

func EncryptMessage(message []byte, cert *x509.Certificate) ([]byte, error) {
	publicKey := cert.PublicKey.(*rsa.PublicKey)
	hash := sha512.New()

	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, publicKey, message, nil)
	if err != nil {
		return nil, err
	}

	return ciphertext, nil
}

func DecryptMessage(ciphertext []byte, key *rsa.PrivateKey) ([]byte, error) {
	hash := sha512.New()

	message, err := rsa.DecryptOAEP(hash, rand.Reader, key, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func GetLastBit(n int) int {
	if n == 0 {
		return 0
	}

	for n > 1 {
		n >>= 1
	}

	return n
}

func GetFirstBit(n, max int) int {
	bitCount := GetBitCount(n)
	maxBitCount := GetBitCount(max)
	if maxBitCount > bitCount {
		return 0
	}
	return (n & (1 << (bitCount - 1))) >> (bitCount - 1)
}

func GetBitCount(n int) int {
	bits := 0
	for n > 0 {
		bits++
		n >>= 1
	}
	return bits
}

func GetMasks(bitCount int) (int, int, int) {
	allMask := 0b1
	lastMask := 0b1
	firstMask := 0b1

	for i := 1; i < bitCount; i++ {
		allMask = (allMask << 1) | 1
		lastMask = lastMask >> 1
		firstMask = firstMask << 1
	}

	return allMask, lastMask, firstMask
}
