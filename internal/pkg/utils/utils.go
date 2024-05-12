package utils

import (
	"crypto/tls"
	"crypto/x509"
	"math/rand"
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

func GetTLSConfig(certPath string, keyPath string, caPath string) (*tls.Config, error) {
	cer, err := GetX509Certificate(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	caCertPool, err := GetCAPool(caPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cer},
		RootCAs:      caCertPool,
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}

func GenerateRandomId() uint64 {
	return rand.Uint64()
}
