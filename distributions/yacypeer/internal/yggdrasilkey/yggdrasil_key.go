// Package yggdrasilkey keeps the private key of the Yggdrasil node of the peer, in one file of a
// directory. It makes the key on the first start and reads the same key on every later start,
// so the peer keeps its Yggdrasil address across restarts.
package yggdrasilkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

const (
	fileName          = "private-key.pem"
	pemBlockType      = "PRIVATE KEY"
	directoryFileMode = 0o700
	keyFileMode       = 0o600
)

var errNotAKey = errors.New("the Yggdrasil key file holds no ed25519 private key")

func Settle(directory string) (ed25519.PrivateKey, error) {
	if err := os.MkdirAll(directory, directoryFileMode); err != nil {
		return nil, fmt.Errorf("make the Yggdrasil key directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open the Yggdrasil key directory: %w", err)
	}
	defer func() { _ = root.Close() }()
	kept, err := root.ReadFile(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return madeIn(root)
	}
	if err != nil {
		return nil, fmt.Errorf("read the Yggdrasil key: %w", err)
	}
	return keyFrom(kept)
}

func madeIn(root *os.Root) (ed25519.PrivateKey, error) {
	_, key, _ := ed25519.GenerateKey(rand.Reader)
	encoded, _ := x509.MarshalPKCS8PrivateKey(key)
	block := pem.EncodeToMemory(&pem.Block{Type: pemBlockType, Bytes: encoded})
	if err := root.WriteFile(fileName, block, keyFileMode); err != nil {
		return nil, fmt.Errorf("keep the Yggdrasil key: %w", err)
	}
	return key, nil
}

func keyFrom(kept []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(kept)
	if block == nil || block.Type != pemBlockType {
		return nil, errNotAKey
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errNotAKey, err)
	}
	key, isEd25519 := parsed.(ed25519.PrivateKey)
	if !isEd25519 {
		return nil, errNotAKey
	}
	return key, nil
}
