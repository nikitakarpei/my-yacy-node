// Package peerkey keeps the peer key of the peer, in one file of a directory. It makes the key
// on the first start and reads the same key on every later start, because the peer hash and
// the data of the node derive from it.
package peerkey

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	fileName          = "peer-key"
	directoryFileMode = 0o700
	keyFileMode       = 0o600
)

func Settle(
	ctx context.Context,
	directory string,
	newKey func() (yacymodel.PeerKey, error),
) (yacymodel.PeerKey, error) {
	if err := os.MkdirAll(directory, directoryFileMode); err != nil {
		return yacymodel.PeerKey{}, fmt.Errorf("make the peer key directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return yacymodel.PeerKey{}, fmt.Errorf("open the peer key directory: %w", err)
	}
	defer func() { _ = root.Close() }()
	kept, err := root.ReadFile(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return madeIn(ctx, root, newKey)
	}
	if err != nil {
		return yacymodel.PeerKey{}, fmt.Errorf("read the peer key: %w", err)
	}
	key, err := yacymodel.ParsePeerKey(strings.TrimSpace(string(kept)))
	if err != nil {
		return yacymodel.PeerKey{}, fmt.Errorf("read the peer key: %w", err)
	}
	return key, nil
}

func madeIn(
	ctx context.Context,
	root *os.Root,
	newKey func() (yacymodel.PeerKey, error),
) (yacymodel.PeerKey, error) {
	slog.InfoContext(ctx, "peer key solving started")
	started := time.Now()
	key, err := newKey()
	if err != nil {
		return yacymodel.PeerKey{}, fmt.Errorf("make the peer key: %w", err)
	}
	if err := root.WriteFile(fileName, []byte(key.String()+"\n"), keyFileMode); err != nil {
		return yacymodel.PeerKey{}, fmt.Errorf("keep the peer key: %w", err)
	}
	slog.InfoContext(ctx, "peer key made",
		slog.String("peer", key.Identity().Hash().String()),
		slog.Duration("duration", time.Since(started)),
	)
	return key, nil
}
