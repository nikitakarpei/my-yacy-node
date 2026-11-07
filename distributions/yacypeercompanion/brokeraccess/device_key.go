package brokeraccess

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"
)

const (
	certificateName = "yacypeer companion"
	certificateLife = 100 * 365 * 24 * time.Hour
)

var errNotAnECKey = errors.New("the device key is not an EC key")

type DeviceKey interface {
	PublicKeyInfo() []byte
	SignDigest(digest []byte) ([]byte, error)
}

type keystoreSigner struct {
	key       DeviceKey
	publicKey *ecdsa.PublicKey
}

func SignerOf(key DeviceKey) (crypto.Signer, error) {
	publicKey, err := x509.ParsePKIXPublicKey(key.PublicKeyInfo())
	if err != nil {
		return nil, fmt.Errorf("read the device key: %w", err)
	}
	ecKey, isEC := publicKey.(*ecdsa.PublicKey)
	if !isEC {
		return nil, errNotAnECKey
	}
	return keystoreSigner{key: key, publicKey: ecKey}, nil
}

func (s keystoreSigner) Public() crypto.PublicKey {
	return s.publicKey
}

func (s keystoreSigner) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	signature, err := s.key.SignDigest(digest)
	if err != nil {
		return nil, fmt.Errorf("sign with the device key: %w", err)
	}
	return signature, nil
}

func certificateOf(key DeviceKey) (tls.Certificate, error) {
	signer, err := SignerOf(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: certificateName},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(certificateLife),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, signer.Public(), signer)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("make the device certificate: %w", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: signer}, nil
}
