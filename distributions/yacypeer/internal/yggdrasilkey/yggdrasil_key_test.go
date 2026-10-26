package yggdrasilkey_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/yggdrasilkey"
)

func TestSettleKeepsTheKeyAcrossStarts(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "yggdrasil")
	first, err := yggdrasilkey.Settle(directory)
	if err != nil {
		t.Fatalf("first Settle: %v", err)
	}

	second, err := yggdrasilkey.Settle(directory)
	if err != nil {
		t.Fatalf("second Settle: %v", err)
	}
	if !second.Equal(first) {
		t.Error("the second start read a different key")
	}
}

func TestSettleMakesADifferentKeyForEachPeer(t *testing.T) {
	first, err := yggdrasilkey.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	second, err := yggdrasilkey.Settle(t.TempDir())
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if first.Equal(second) {
		t.Error("two peers got the same key")
	}
}

func TestSettleLetsOnlyTheOwnerReadTheKey(t *testing.T) {
	directory := t.TempDir()
	if _, err := yggdrasilkey.Settle(directory); err != nil {
		t.Fatalf("Settle: %v", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("directory holds %v (%v), want one file", entries, err)
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("stat the key file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("key file mode = %o, want 600", mode)
	}
}

func TestSettleRefusesAFileThatHoldsNoKey(t *testing.T) {
	directory := t.TempDir()
	keepInOnlyFile(t, directory, []byte("not a key\n"))

	if _, err := yggdrasilkey.Settle(directory); err == nil {
		t.Error("Settle accepted a file that holds no key")
	}
}

func TestSettleRefusesAKeyThatIsNotEd25519(t *testing.T) {
	directory := t.TempDir()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("make an ecdsa key: %v", err)
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("encode the ecdsa key: %v", err)
	}
	keepInOnlyFile(
		t,
		directory,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}),
	)

	if _, err := yggdrasilkey.Settle(directory); err == nil {
		t.Error("Settle accepted an ecdsa key")
	}
}

func TestSettleRefusesAPEMBlockThatIsNoPKCS8Key(t *testing.T) {
	directory := t.TempDir()
	keepInOnlyFile(
		t,
		directory,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte{1}}),
	)

	if _, err := yggdrasilkey.Settle(directory); err == nil {
		t.Error("Settle accepted a PEM block that holds no PKCS8 key")
	}
}

func keepInOnlyFile(t *testing.T, directory string, content []byte) {
	t.Helper()
	if _, err := yggdrasilkey.Settle(directory); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("directory holds %v (%v), want one file", entries, err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, entries[0].Name()),
		content,
		0o600,
	); err != nil {
		t.Fatalf("overwrite the key file: %v", err)
	}
}
