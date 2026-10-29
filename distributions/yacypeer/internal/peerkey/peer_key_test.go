package peerkey_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/peerkey"
)

var errNoEntropy = errors.New("no entropy")

func solvedKey(context.Context, time.Duration) (yacymodel.PeerKey, error) {
	return peerkeytest.Key(0), nil
}

func unsolvableKey(context.Context, time.Duration) (yacymodel.PeerKey, error) {
	return yacymodel.PeerKey{}, errNoEntropy
}

func TestSettleMakesAKeyOnTheFirstStartAndKeepsIt(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "data")

	first, err := peerkey.Settle(t.Context(), directory, time.Minute, solvedKey)
	if err != nil {
		t.Fatalf("first Settle: %v", err)
	}
	second, err := peerkey.Settle(t.Context(), directory, time.Minute, unsolvableKey)
	if err != nil {
		t.Fatalf("second Settle: %v", err)
	}
	made := peerkeytest.SecretTextOf(peerkeytest.Key(0))
	if peerkeytest.SecretTextOf(first) != made || peerkeytest.SecretTextOf(second) != made {
		t.Errorf("keys = %v and %v, want the made key twice", first, second)
	}
}

func TestSettleLetsOnlyTheOwnerReadTheKey(t *testing.T) {
	directory := t.TempDir()
	if _, err := peerkey.Settle(t.Context(), directory, time.Minute, solvedKey); err != nil {
		t.Fatalf("Settle: %v", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("directory holds %v (%v), want one file", entries, err)
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o400 {
		t.Errorf("mode = %v, want 0400", info.Mode().Perm())
	}
}

func TestSettleReplacesADraftLeftByAnEarlierStart(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "peer.key.draft"),
		[]byte("cut"),
		0o400,
	); err != nil {
		t.Fatalf("write draft: %v", err)
	}

	key, err := peerkey.Settle(t.Context(), directory, time.Minute, solvedKey)
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	kept, err := peerkey.Settle(t.Context(), directory, time.Minute, unsolvableKey)
	if err != nil || peerkeytest.SecretTextOf(kept) != peerkeytest.SecretTextOf(key) {
		t.Errorf("kept key = %v (%v), want %v", kept, err, key)
	}
}

func TestSettleFails(t *testing.T) {
	for name, directoryFor := range map[string]func(t *testing.T) string{
		"directory is a file": func(t *testing.T) string {
			file := filepath.Join(t.TempDir(), "file")
			if err := os.WriteFile(file, nil, 0o600); err != nil {
				t.Fatalf("write file: %v", err)
			}
			return file
		},
		"kept key is bad": func(t *testing.T) string {
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, "peer.key"), []byte("bad\n"), 0o600); err != nil {
				t.Fatalf("write key: %v", err)
			}
			return directory
		},
		"kept key is unreadable": func(t *testing.T) string {
			directory := t.TempDir()
			if err := os.Mkdir(filepath.Join(directory, "peer.key"), 0o700); err != nil {
				t.Fatalf("make directory: %v", err)
			}
			return directory
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := peerkey.Settle(
				t.Context(),
				directoryFor(t),
				time.Minute,
				solvedKey,
			); err == nil {
				t.Error("Settle = nil, want an error")
			}
		})
	}
}

func TestSettleFailsWhenNoKeyCanBeMade(t *testing.T) {
	_, err := peerkey.Settle(t.Context(), t.TempDir(), time.Minute, unsolvableKey)
	if !errors.Is(err, errNoEntropy) {
		t.Errorf("Settle = %v, want %v", err, errNoEntropy)
	}
}

func TestSettleSearchesANewKeyForTheSearchTime(t *testing.T) {
	var searchedFor time.Duration
	searchedKey := func(ctx context.Context, searchTime time.Duration) (yacymodel.PeerKey, error) {
		searchedFor = searchTime

		return solvedKey(ctx, searchTime)
	}

	if _, err := peerkey.Settle(t.Context(), t.TempDir(), 3*time.Minute, searchedKey); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if searchedFor != 3*time.Minute {
		t.Errorf("search time = %v, want 3m", searchedFor)
	}
}
