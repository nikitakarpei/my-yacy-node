package visitlinksecret_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/visitlinksecret"
)

func TestSettleMakesASecretOnTheFirstStart(t *testing.T) {
	secret, err := visitlinksecret.Settle(filepath.Join(t.TempDir(), "visitcrawl"))
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if len(secret) != 64 {
		t.Errorf("secret = %q, want 64 hexadecimal characters", secret)
	}
}

func TestSettleKeepsTheSecretAcrossStarts(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "visitcrawl")
	first, err := visitlinksecret.Settle(directory)
	if err != nil {
		t.Fatalf("first Settle: %v", err)
	}

	second, err := visitlinksecret.Settle(directory)
	if err != nil {
		t.Fatalf("second Settle: %v", err)
	}
	if second != first {
		t.Errorf("second secret = %q, want the first %q", second, first)
	}
}

func TestSettleMakesADifferentSecretForEachPeer(t *testing.T) {
	first, err := visitlinksecret.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	second, err := visitlinksecret.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if first == second {
		t.Error("two peers got the same secret")
	}
}

func TestSettleLetsOnlyTheOwnerReadTheSecret(t *testing.T) {
	directory := t.TempDir()
	if _, err := visitlinksecret.Settle(directory); err != nil {
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
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestSettleFailsWhenTheDirectoryIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := visitlinksecret.Settle(file); err == nil {
		t.Error("Settle succeeded inside a file")
	}
}

func TestSettleFailsWhenTheSecretCannotBeRead(t *testing.T) {
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "link-secret"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := visitlinksecret.Settle(directory); err == nil {
		t.Error("Settle succeeded with a directory in place of the secret")
	}
}
