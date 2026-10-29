package yacymodel_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
)

func TestPeerKeyTextRoundTrips(t *testing.T) {
	key := peerkeytest.Key(0)

	parsed, err := yacymodel.ParsePeerKey(key.String())
	if err != nil {
		t.Fatalf("ParsePeerKey(%q): %v", key, err)
	}
	if parsed.String() != key.String() {
		t.Errorf("String = %q, want %q", parsed, key)
	}
	if parsed.Identity() != key.Identity() {
		t.Errorf("Identity = %v, want %v", parsed.Identity(), key.Identity())
	}
}

func TestPeerKeysOfOnePeerDeriveOneHashAndOthersAnother(t *testing.T) {
	first, second := peerkeytest.Key(0), peerkeytest.Key(1)

	if first.Hash() == second.Hash() {
		t.Errorf("two keys derive one hash %s", first.Hash())
	}
	if first.Hash() != peerkeytest.Key(0).Hash() {
		t.Error("one key derives two hashes")
	}
}

func TestParsePeerKeyRejects(t *testing.T) {
	parts := strings.Split(peerkeytest.Key(0).String(), ":")
	for name, text := range map[string]string{
		"empty":                       "",
		"two parts":                   parts[0] + ":" + parts[1],
		"four parts":                  peerkeytest.Key(0).String() + ":" + parts[2],
		"short operational key":       parts[0][1:] + ":" + parts[1] + ":" + parts[2],
		"bad symbol":                  "*" + parts[0][1:] + ":" + parts[1] + ":" + parts[2],
		"short recovery key":          parts[0] + ":" + parts[1][1:] + ":" + parts[2],
		"short nonce":                 parts[0] + ":" + parts[1] + ":" + parts[2][1:],
		"nonce of another key":        parts[0] + ":" + parts[1] + ":" + nonceTextOf(peerkeytest.Key(1)),
		"second text of one nonce":    parts[0] + ":" + parts[1] + ":" + withLastSymbolRaised(parts[2]),
		"recovery key of another key": parts[0] + ":" + recoveryTextOf(peerkeytest.Key(1)) + ":" + parts[2],
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := yacymodel.ParsePeerKey(text); !errors.Is(err, yacymodel.ErrBadPeerKey) {
				t.Errorf("ParsePeerKey(%q) = %v, want ErrBadPeerKey", text, err)
			}
		})
	}
}

func TestParsePeerIdentityAcceptsTheIdentityOfAKey(t *testing.T) {
	identity := peerkeytest.Key(0).Identity()

	parsed, err := yacymodel.ParsePeerIdentity(identity.String())
	if err != nil {
		t.Fatalf("ParsePeerIdentity(%q): %v", identity, err)
	}
	if parsed != identity || parsed.Hash() != identity.Hash() {
		t.Errorf("ParsePeerIdentity = %v, want %v", parsed, identity)
	}
}

func TestParsePeerIdentityRejects(t *testing.T) {
	parts := strings.Split(peerkeytest.Key(0).Identity().String(), ":")
	otherParts := strings.Split(peerkeytest.Key(1).Identity().String(), ":")
	for name, text := range map[string]string{
		"empty":                        "",
		"two parts":                    parts[0] + ":" + parts[1],
		"short operational key":        parts[0][1:] + ":" + parts[1] + ":" + parts[2],
		"short recovery commitment":    parts[0] + ":" + parts[1][1:] + ":" + parts[2],
		"bad nonce symbol":             parts[0] + ":" + parts[1] + ":" + "*" + parts[2][1:],
		"nonce of another identity":    parts[0] + ":" + parts[1] + ":" + otherParts[2],
		"operational key of another":   otherParts[0] + ":" + parts[1] + ":" + parts[2],
		"second text of one key":       withLastSymbolRaised(parts[0]) + ":" + parts[1] + ":" + parts[2],
		"recovery commitment of other": parts[0] + ":" + otherParts[1] + ":" + parts[2],
	} {
		t.Run(name, func(t *testing.T) {
			_, err := yacymodel.ParsePeerIdentity(text)
			if !errors.Is(err, yacymodel.ErrBadPeerIdentity) {
				t.Errorf("ParsePeerIdentity(%q) = %v, want ErrBadPeerIdentity", text, err)
			}
		})
	}
}

const quicklySolvedEntropy = "e09761483d092260bb47418d6081c3ea30b640186d15ed263954bdcfeb652b9c" +
	"27eac2f3aada231935a88b823de9fde7ee18e3a2baed27ac459cb551d4bb1c7e" +
	"aaea5ef9eb67bebb2037b980062a2e12"

func TestGeneratePeerKeySolvesTheWorkOfANewIdentity(t *testing.T) {
	entropy, err := hex.DecodeString(quicklySolvedEntropy)
	if err != nil {
		t.Fatalf("decode entropy: %v", err)
	}

	key, err := yacymodel.GeneratePeerKey(t.Context(), bytes.NewReader(entropy))
	if err != nil {
		t.Fatalf("GeneratePeerKey: %v", err)
	}

	parsed, err := yacymodel.ParsePeerKey(key.String())
	if err != nil {
		t.Fatalf("ParsePeerKey(%q): %v", key, err)
	}
	if _, err := yacymodel.ParsePeerIdentity(parsed.Identity().String()); err != nil {
		t.Fatalf("ParsePeerIdentity(%q): %v", parsed.Identity(), err)
	}
}

func TestGeneratePeerKeyFailsWithoutEntropy(t *testing.T) {
	for name, entropy := range map[string]io.Reader{
		"none":                  bytes.NewReader(nil),
		"operational key alone": bytes.NewReader(make([]byte, 32)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := yacymodel.GeneratePeerKey(t.Context(), entropy); err == nil {
				t.Error("GeneratePeerKey = nil, want an error")
			}
		})
	}
}

func TestGeneratePeerKeyStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := yacymodel.GeneratePeerKey(ctx, bytes.NewReader(make([]byte, 80)))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("GeneratePeerKey = %v, want %v", err, context.Canceled)
	}
}

func nonceTextOf(key yacymodel.PeerKey) string {
	return strings.Split(key.String(), ":")[2]
}

func recoveryTextOf(key yacymodel.PeerKey) string {
	return strings.Split(key.String(), ":")[1]
}

func withLastSymbolRaised(text string) string {
	last := strings.IndexByte(yacymodel.Alphabet, text[len(text)-1])

	return text[:len(text)-1] + string(yacymodel.Alphabet[last+1])
}
