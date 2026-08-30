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

func TestHashKeysCarryTheHashText(t *testing.T) {
	word := yacymodel.WordHash("keyword")
	parts := vault.SingleKey(hashkeypart.Hash)

	if !bytes.Equal(
		parts.Key(word).Bytes(),
		vault.SingleKey(vault.TextKeyPart).Key(word.String()).Bytes(),
	) {
		t.Fatalf("key(%s) differs from the text key of its string form", word)
	}

	decoded, err := parts.PartsOf(parts.Key(word).Bytes())
	if err != nil {
		t.Fatalf("PartsOf(%s) failed: %v", word, err)
	}
	if decoded != word {
		t.Fatalf("PartsOf = %s, want %s", decoded, word)
	}
}

func TestURLHashKeysCarryTheHashText(t *testing.T) {
	address, err := neturl.Parse("http://example.org/page")
	if err != nil {
		t.Fatalf("parse address: %v", err)
	}
	url := yacymodel.URLNormalformOf(address).Hash()
	parts := vault.SingleKey(hashkeypart.URLHash)

	if !bytes.Equal(
		parts.Key(url).Bytes(),
		vault.SingleKey(vault.TextKeyPart).Key(url.String()).Bytes(),
	) {
		t.Fatalf("key(%s) differs from the text key of its string form", url)
	}

	decoded, err := parts.PartsOf(parts.Key(url).Bytes())
	if err != nil {
		t.Fatalf("PartsOf(%s) failed: %v", url, err)
	}
	if decoded != url {
		t.Fatalf("PartsOf = %s, want %s", decoded, url)
	}
}

func TestHashKeysFailWhenTheTextIsNoHash(t *testing.T) {
	notAHash := vault.SingleKey(vault.TextKeyPart).Key("too short")

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
