// Package brokerkey keeps the TLS key of the broker of the peer in one file of a directory. It
// makes the key on the first start and reads the same key on every later start, so the key hash
// that each paired companion checks stays valid across restarts.
package brokerkey

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"math/big"
	"os"
	"time"
)

const (
	fileName          = "broker-key.pem"
	pemType           = "EC PRIVATE KEY"
	directoryFileMode = 0o700
	keyFileMode       = 0o600
	certificateName   = "yacypeer broker"
	certificateLife   = 100 * 365 * 24 * time.Hour
)

var errUnreadableKey = errors.New("the broker key file holds no key")

func Settle(directory string) (tls.Certificate, error) {
	key, err := keptKeyIn(directory)
	if err != nil {
		return tls.Certificate{}, err
	}
	return certificateFor(key)
}

func keptKeyIn(directory string) (*ecdsa.PrivateKey, error) {
	if err := os.MkdirAll(directory, directoryFileMode); err != nil {
		return nil, fmt.Errorf("make the broker key directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open the broker key directory: %w", err)
	}
	defer closeDirectory(root)
	kept, err := root.ReadFile(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return madeIn(root)
	}
	if err != nil {
		return nil, fmt.Errorf("read the broker key: %w", err)
	}
	return keyFrom(kept)
}

func closeDirectory(root *os.Root) {
	if err := root.Close(); err != nil {
		slog.WarnContext(
			context.Background(),
			"broker key directory not closed",
			slog.Any("error", err),
		)
	}
}

func madeIn(root *os.Root) (*ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("make the broker key: %w", err)
	}
	encoded, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("encode the broker key: %w", err)
	}
	block := pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: encoded})
	if err := root.WriteFile(fileName, block, keyFileMode); err != nil {
		return nil, fmt.Errorf("keep the broker key: %w", err)
	}
	return key, nil
}

func keyFrom(kept []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(kept)
	if block == nil || block.Type != pemType {
		return nil, errUnreadableKey
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("read the broker key: %w", err)
	}
	return key, nil
}

func certificateFor(key *ecdsa.PrivateKey) (tls.Certificate, error) {
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: certificateName},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(certificateLife),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("make the broker certificate: %w", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("read the broker certificate: %w", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, nil
}
