package yacymodel_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
)

func TestPeerKeyTextRoundTrips(t *testing.T) {
	key := peerkeytest.Key(0)

	parsed, err := yacymodel.ParsePeerKey(peerkeytest.SecretTextOf(key))
	if err != nil {
		t.Fatalf("ParsePeerKey(the text of %v): %v", key, err)
	}
	if peerkeytest.SecretTextOf(parsed) != peerkeytest.SecretTextOf(key) {
		t.Errorf("SecretText of %v differs from the parsed text", key)
	}
	if parsed.Identity() != key.Identity() {
		t.Errorf("Identity = %v, want %v", parsed.Identity(), key.Identity())
	}
}

func TestPeerKeyPrintsNoSecret(t *testing.T) {
	key := peerkeytest.Key(0)
	printed := fmt.Sprintf("%v %+v %#v %s %q", key, key, key, key, key)

	for _, secretPart := range strings.Split(peerkeytest.SecretTextOf(key), ":")[:2] {
		if strings.Contains(printed, secretPart) {
			t.Errorf("printed key %q holds the secret part %q", printed, secretPart)
		}
	}
	if !strings.Contains(printed, key.Hash().String()) {
		t.Errorf("printed key %q, want its hash %s", printed, key.Hash())
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
	parts := strings.Split(peerkeytest.SecretTextOf(peerkeytest.Key(0)), ":")
	for name, text := range map[string]string{
		"empty":                       "",
		"two parts":                   parts[0] + ":" + parts[1],
		"four parts":                  peerkeytest.SecretTextOf(peerkeytest.Key(0)) + ":" + parts[2],
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
				t.Errorf("ParsePeerKey = %v, want ErrBadPeerKey", err)
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

	key, err := yacymodel.GeneratePeerKey(
		t.Context(),
		bytes.NewReader(entropy),
		time.Minute,
		expiredClock{},
	)
	if err != nil {
		t.Fatalf("GeneratePeerKey: %v", err)
	}

	parsed, err := yacymodel.ParsePeerKey(peerkeytest.SecretTextOf(key))
	if err != nil {
		t.Fatalf("ParsePeerKey(the text of %v): %v", key, err)
	}
	if _, err := yacymodel.ParsePeerIdentity(parsed.Identity().String()); err != nil {
		t.Fatalf("ParsePeerIdentity(%q): %v", parsed.Identity(), err)
	}
	if parsed.Work() < 28 {
		t.Errorf("Work = %d, want at least 28", parsed.Work())
	}
}

func TestGeneratePeerKeyFailsWithoutEntropy(t *testing.T) {
	for name, entropy := range map[string]io.Reader{
		"none":                  bytes.NewReader(nil),
		"operational key alone": bytes.NewReader(make([]byte, 32)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := yacymodel.GeneratePeerKey(
				t.Context(),
				entropy,
				time.Minute,
				expiredClock{},
			); err == nil {
				t.Error("GeneratePeerKey = nil, want an error")
			}
		})
	}
}

func TestGeneratePeerKeyStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := yacymodel.GeneratePeerKey(
		ctx,
		bytes.NewReader(make([]byte, 80)),
		time.Minute,
		newHeldClock(),
	)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("GeneratePeerKey = %v, want %v", err, context.Canceled)
	}
}

func TestGeneratePeerKeyKeepsSearchingUntilItsSearchTimeIsOver(t *testing.T) {
	entropy, err := hex.DecodeString(quicklySolvedEntropy)
	if err != nil {
		t.Fatalf("decode entropy: %v", err)
	}
	clock := newHeldClock()
	go clock.expireWhenHeld()

	key, err := yacymodel.GeneratePeerKey(t.Context(), bytes.NewReader(entropy), time.Minute, clock)
	if err != nil {
		t.Fatalf("GeneratePeerKey: %v", err)
	}
	if clock.timeout != time.Minute {
		t.Errorf("search time = %v, want %v", clock.timeout, time.Minute)
	}
	if key.Work() < 28 {
		t.Errorf("Work = %d, want at least 28", key.Work())
	}
}

type expiredClock struct{}

func (expiredClock) After(_ time.Duration, expire func()) (stop func()) {
	expire()

	return func() {}
}

type heldClock struct {
	timeout time.Duration
	expire  func()
	held    chan struct{}
}

func newHeldClock() *heldClock {
	return &heldClock{held: make(chan struct{})}
}

func (c *heldClock) After(timeout time.Duration, expire func()) (stop func()) {
	c.timeout, c.expire = timeout, expire
	close(c.held)

	return func() {}
}

func (c *heldClock) expireWhenHeld() {
	<-c.held
	c.expire()
}

func nonceTextOf(key yacymodel.PeerKey) string {
	return strings.Split(peerkeytest.SecretTextOf(key), ":")[2]
}

func recoveryTextOf(key yacymodel.PeerKey) string {
	return strings.Split(peerkeytest.SecretTextOf(key), ":")[1]
}

func withLastSymbolRaised(text string) string {
	last := strings.IndexByte(yacymodel.Alphabet, text[len(text)-1])

	return text[:len(text)-1] + string(yacymodel.Alphabet[last+1])
}
