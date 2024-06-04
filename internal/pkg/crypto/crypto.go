package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"time"
)

func GenerateRSAKey(bitSize int) (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, bitSize)
}

func EncodeRSAKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)
}

func GenerateCACertificate() *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(2024),
		Subject: pkix.Name{
			Organization: []string{"Trust"},
			Country:      []string{"UA"},
			Province:     []string{""},
			Locality:     []string{"Kyiv"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
}

func EncodeCertificate(cert *x509.Certificate, parent *x509.Certificate, key *rsa.PrivateKey, parentKey *rsa.PrivateKey) ([]byte, error) {
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, parent, &key.PublicKey, parentKey)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		},
	), nil
}

func GenerateCertificate(serial int64, a, b, c, d byte) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			Organization: []string{"Trust"},
			Country:      []string{"UA"},
			Province:     []string{""},
			Locality:     []string{"Kyiv"},
		},
		IPAddresses: []net.IP{net.IPv4(a, b, c, d), net.IPv6loopback},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}
}

func GenerateAESKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
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

func EncryptMessageAES(message []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(message))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCFBEncrypter(block, iv)
	mode.XORKeyStream(ciphertext[aes.BlockSize:], message)

	return ciphertext, nil
}

func DecryptMessageAES(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, io.ErrUnexpectedEOF
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCFBDecrypter(block, iv)
	mode.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

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
