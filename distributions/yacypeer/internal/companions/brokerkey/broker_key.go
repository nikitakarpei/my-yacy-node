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
	draftFileName     = fileName + ".draft"
	pemType           = "EC PRIVATE KEY"
	directoryFileMode = 0o700
	keyFileMode       = 0o600
	certificateName   = "yacypeer broker"
	certificateLife   = 100 * 365 * 24 * time.Hour
)

var errUnreadableKey = errors.New("the broker key file holds no key")

func Settle(directory string) (tls.Certificate, error) {
	root, err := openedDirectory(directory)
	if err != nil {
		return tls.Certificate{}, err
	}
	defer closeDirectory(root)
	if !holdsKey(root) {
		key, err := randomKey()
		if err != nil {
			return tls.Certificate{}, err
		}
		if err := keep(root, key); err != nil {
			return tls.Certificate{}, err
		}
	}
	key, err := keptKeyIn(root)
	if err != nil {
		return tls.Certificate{}, err
	}
	return certificateFor(key)
}

func openedDirectory(directory string) (*os.Root, error) {
	if err := os.MkdirAll(directory, directoryFileMode); err != nil {
		return nil, fmt.Errorf("make the broker key directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open the broker key directory: %w", err)
	}
	return root, nil
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

func holdsKey(root *os.Root) bool {
	_, err := root.Stat(fileName)
	return !errors.Is(err, fs.ErrNotExist)
}

func randomKey() (*ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("make the broker key: %w", err)
	}
	return key, nil
}

func keep(root *os.Root, key *ecdsa.PrivateKey) error {
	encoded, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("encode the broker key: %w", err)
	}
	block := pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: encoded})
	if err := root.WriteFile(draftFileName, block, keyFileMode); err != nil {
		return fmt.Errorf("keep the broker key: %w", err)
	}
	if err := root.Rename(draftFileName, fileName); err != nil {
		return fmt.Errorf("keep the broker key: %w", err)
	}
	return nil
}

func keptKeyIn(root *os.Root) (*ecdsa.PrivateKey, error) {
	kept, err := root.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("read the broker key: %w", err)
	}
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
	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&key.PublicKey,
		key,
	)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("make the broker certificate: %w", err)
	}
	leaf, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("read the broker certificate: %w", err)
	}
	return tls.Certificate{Certificate: [][]byte{certificateDER}, PrivateKey: key, Leaf: leaf}, nil
}
