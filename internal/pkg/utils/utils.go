package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"math"
	"math/big"
	"os"
)

func GetX509Certificate(certPath string, keyPath string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certPath, keyPath)
}

func GetCAPool(caPath string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	return caCertPool, nil
}

func GetTLSConfig(certPath string, keyPath string, caPath *string) (*tls.Config, error) {
	cer, err := GetX509Certificate(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	config := tls.Config{
		Certificates:       []tls.Certificate{cer},
		InsecureSkipVerify: caPath == nil,
	}

	if caPath != nil {
		caCertPool, err := GetCAPool(*caPath)
		if err != nil {
			return nil, err
		}

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
