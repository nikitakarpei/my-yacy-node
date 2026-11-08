package brokerkey_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
)

func TestSettleKeepsTheKeyAcrossStarts(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "companions")
	first, err := brokerkey.Settle(directory)
	if err != nil {
		t.Fatalf("first Settle: %v", err)
	}

	second, err := brokerkey.Settle(directory)
	if err != nil {
		t.Fatalf("second Settle: %v", err)
	}
	if !bytes.Equal(first.Leaf.RawSubjectPublicKeyInfo, second.Leaf.RawSubjectPublicKeyInfo) {
		t.Error("the second start has another key")
	}
}

func TestSettleMakesADifferentKeyForEachPeer(t *testing.T) {
	first, err := brokerkey.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	second, err := brokerkey.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if bytes.Equal(first.Leaf.RawSubjectPublicKeyInfo, second.Leaf.RawSubjectPublicKeyInfo) {
		t.Error("two peers got the same key")
	}
}

func TestSettleLetsOnlyTheOwnerReadTheKey(t *testing.T) {
	directory := t.TempDir()
	if _, err := brokerkey.Settle(directory); err != nil {
		t.Fatalf("Settle: %v", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("directory holds %v (%v), want one file", entries, err)
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestSettleRefusesAFileThatHoldsNoKey(t *testing.T) {
	directory := t.TempDir()
	if _, err := brokerkey.Settle(directory); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("directory holds %v (%v), want one file", entries, err)
	}
	keyFile := filepath.Join(directory, entries[0].Name())
	if err := os.WriteFile(keyFile, []byte("not a key"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := brokerkey.Settle(directory); err == nil {
		t.Error("want an error for a file that holds no key")
	}
}
