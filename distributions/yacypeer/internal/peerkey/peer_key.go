// Package peerkey keeps the peer key of the peer, in one file of a directory. It makes the key
// on the first start and reads the same key on every later start, because the peer hash and
// the data of the node derive from it.
package peerkey

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	fileName          = "peer.key"
	draftFileName     = "peer.key.draft"
	directoryFileMode = 0o700
	keyFileMode       = 0o400
)

func Settle(
	ctx context.Context,
	directory string,
	newKey func(context.Context) (yacymodel.PeerKey, error),
) (yacymodel.PeerKey, error) {
	root, err := openedDirectory(directory)
	if err != nil {
		return yacymodel.PeerKey{}, err
	}
	defer func() { _ = root.Close() }()
	keptKey, err := keptKeyIn(root)
	if err != nil {
		return yacymodel.PeerKey{}, err
	}
	if key, kept := keptKey.Get(); kept {
		return key, nil
	}
	key, err := solvedKey(ctx, newKey)
	if err != nil {
		return yacymodel.PeerKey{}, err
	}

	return key, keep(root, key)
}

func openedDirectory(directory string) (*os.Root, error) {
	if err := os.MkdirAll(directory, directoryFileMode); err != nil {
		return nil, fmt.Errorf("make the peer key directory: %w", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open the peer key directory: %w", err)
	}

	return root, nil
}

func keptKeyIn(root *os.Root) (yacymodel.Optional[yacymodel.PeerKey], error) {
	kept, err := root.ReadFile(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return yacymodel.None[yacymodel.PeerKey](), nil
	}
	if err != nil {
		return yacymodel.None[yacymodel.PeerKey](), fmt.Errorf("read the peer key: %w", err)
	}
	key, err := yacymodel.ParsePeerKey(strings.TrimSpace(string(kept)))
	if err != nil {
		return yacymodel.None[yacymodel.PeerKey](), fmt.Errorf("read the peer key: %w", err)
	}

	return yacymodel.Some(key), nil
}

func solvedKey(
	ctx context.Context,
	newKey func(context.Context) (yacymodel.PeerKey, error),
) (yacymodel.PeerKey, error) {
	slog.InfoContext(ctx, "peer key solving started")
	started := time.Now()
	key, err := newKey(ctx)
	if err != nil {
		return yacymodel.PeerKey{}, fmt.Errorf("make the peer key: %w", err)
	}
	slog.InfoContext(ctx, "peer key made",
		slog.String("peer", key.Hash().String()),
		slog.Duration("duration", time.Since(started)),
	)

	return key, nil
}

func keep(root *os.Root, key yacymodel.PeerKey) error {
	if err := root.Remove(draftFileName); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("keep the peer key: %w", err)
	}
	if err := writeDurably(root, draftFileName, keyFileWriterFor(key)); err != nil {
		return fmt.Errorf("keep the peer key: %w", err)
	}
	if err := root.Rename(draftFileName, fileName); err != nil {
		return fmt.Errorf("keep the peer key: %w", err)
	}
	if err := syncDirectory(root); err != nil {
		return fmt.Errorf("keep the peer key: %w", err)
	}

	return nil
}

func keyFileWriterFor(key yacymodel.PeerKey) func(io.Writer) error {
	return func(file io.Writer) error {
		if err := key.WriteSecret(file); err != nil {
			return err
		}
		if _, err := io.WriteString(file, "\n"); err != nil {
			return fmt.Errorf("end the peer key line: %w", err)
		}

		return nil
	}
}

func writeDurably(root *os.Root, name string, writeContent func(io.Writer) error) error {
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, keyFileMode)
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	if err := writeContent(file); err != nil {
		_ = file.Close()

		return fmt.Errorf("write %s: %w", name, err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()

		return fmt.Errorf("sync %s: %w", name, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}

	return nil
}

func syncDirectory(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return fmt.Errorf("open the directory: %w", err)
	}
	defer func() { _ = directory.Close() }()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync the directory: %w", err)
	}

	return nil
}
