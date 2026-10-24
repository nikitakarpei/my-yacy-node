// Package visitlinksecret keeps the secret that signs the visit links of the peer, in one file
// of a directory. It makes the secret on the first start and reads the same secret on every
// later start, so a link that a search page issued stays valid across restarts.
package visitlinksecret

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

const (
	fileName          = "link-secret"
	secretByteLength  = 32
	directoryFileMode = 0o700
	secretFileMode    = 0o600
)

func Settle(directory string) (string, error) {
	if err := os.MkdirAll(directory, directoryFileMode); err != nil {
		return "", fmt.Errorf("make the visit link secret directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return "", fmt.Errorf("open the visit link secret directory: %w", err)
	}
	defer func() { _ = root.Close() }()
	kept, err := root.ReadFile(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return madeIn(root)
	}
	if err != nil {
		return "", fmt.Errorf("read the visit link secret: %w", err)
	}
	return strings.TrimSpace(string(kept)), nil
}

func madeIn(root *os.Root) (string, error) {
	secret := make([]byte, secretByteLength)
	_, _ = rand.Read(secret)
	encoded := hex.EncodeToString(secret)
	if err := root.WriteFile(fileName, []byte(encoded+"\n"), secretFileMode); err != nil {
		return "", fmt.Errorf("keep the visit link secret: %w", err)
	}
	return encoded, nil
}
