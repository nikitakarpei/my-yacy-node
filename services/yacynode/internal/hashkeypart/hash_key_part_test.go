package hashkeypart_test

import (
	"bytes"
	"errors"
	neturl "net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

func TestHashKeysCarryTheHashBytes(t *testing.T) {
	word := yacymodel.WordHash("keyword")
	parts := vault.SingleKey(hashkeypart.Hash)

	if !bytes.Equal(
		parts.Key(word).Bytes(),
		vault.SingleKey(vault.TextKeyPart).Key(string(word.Bytes())).Bytes(),
	) {
		t.Fatalf("key(%s) differs from the key of its byte form", word)
	}

	decoded, err := parts.PartsOf(parts.Key(word).Bytes())
	if err != nil {
		t.Fatalf("PartsOf(%s) failed: %v", word, err)
	}
	if decoded != word {
		t.Fatalf("PartsOf = %s, want %s", decoded, word)
	}
}

func TestURLHashKeysCarryTheHashBytes(t *testing.T) {
	address, err := neturl.Parse("http://example.org/page")
	if err != nil {
		t.Fatalf("parse address: %v", err)
	}
	url := yacymodel.URLNormalformOf(address).Hash()
	parts := vault.SingleKey(hashkeypart.URLHash)

	if !bytes.Equal(
		parts.Key(url).Bytes(),
		vault.SingleKey(vault.TextKeyPart).Key(string(url.Bytes())).Bytes(),
	) {
		t.Fatalf("key(%s) differs from the key of its byte form", url)
	}

	decoded, err := parts.PartsOf(parts.Key(url).Bytes())
	if err != nil {
		t.Fatalf("PartsOf(%s) failed: %v", url, err)
	}
	if decoded != url {
		t.Fatalf("PartsOf = %s, want %s", decoded, url)
	}
}

func TestHashKeysFailWhenTheKeyHoldsNoHash(t *testing.T) {
	notAHash := vault.SingleKey(vault.TextKeyPart).Key("too short for a hash")

	if _, err := vault.SingleKey(hashkeypart.Hash).PartsOf(notAHash.Bytes()); !errors.Is(
		err,
		yacymodel.ErrInvalidHash,
	) {
		t.Fatalf("PartsOf error = %v, want %v", err, yacymodel.ErrInvalidHash)
	}
	if _, err := vault.SingleKey(hashkeypart.URLHash).PartsOf(notAHash.Bytes()); !errors.Is(
		err,
		yacymodel.ErrInvalidHash,
	) {
		t.Fatalf("PartsOf error = %v, want %v", err, yacymodel.ErrInvalidHash)
	}
}
