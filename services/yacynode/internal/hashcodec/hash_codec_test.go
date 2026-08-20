package hashcodec_test

import (
	"bytes"
	"errors"
	neturl "net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
)

func TestHashKeysCarryTheHashText(t *testing.T) {
	word := yacymodel.WordHash("keyword")
	layout := vault.SingleKey(hashcodec.Hash)

	if !bytes.Equal(
		layout.Key(word).Bytes(),
		vault.SingleKey(vault.TextKeyPart).Key(word.String()).Bytes(),
	) {
		t.Fatalf("key(%s) differs from the text key of its string form", word)
	}

	decoded, err := layout.Parts(layout.Key(word).Bytes())
	if err != nil {
		t.Fatalf("Parts(%s) failed: %v", word, err)
	}
	if decoded != word {
		t.Fatalf("Parts = %s, want %s", decoded, word)
	}
}

func TestURLHashKeysCarryTheHashText(t *testing.T) {
	address, err := neturl.Parse("http://example.org/page")
	if err != nil {
		t.Fatalf("parse address: %v", err)
	}
	url := yacymodel.URLNormalformOf(address).Hash()
	layout := vault.SingleKey(hashcodec.URLHash)

	if !bytes.Equal(
		layout.Key(url).Bytes(),
		vault.SingleKey(vault.TextKeyPart).Key(url.String()).Bytes(),
	) {
		t.Fatalf("key(%s) differs from the text key of its string form", url)
	}

	decoded, err := layout.Parts(layout.Key(url).Bytes())
	if err != nil {
		t.Fatalf("Parts(%s) failed: %v", url, err)
	}
	if decoded != url {
		t.Fatalf("Parts = %s, want %s", decoded, url)
	}
}

func TestHashKeysFailWhenTheTextIsNoHash(t *testing.T) {
	notAHash := vault.SingleKey(vault.TextKeyPart).Key("too short")

	if _, err := vault.SingleKey(hashcodec.Hash).Parts(notAHash.Bytes()); !errors.Is(
		err,
		yacymodel.ErrInvalidHash,
	) {
		t.Fatalf("Parts error = %v, want %v", err, yacymodel.ErrInvalidHash)
	}
	if _, err := vault.SingleKey(hashcodec.URLHash).Parts(notAHash.Bytes()); !errors.Is(
		err,
		yacymodel.ErrInvalidHash,
	) {
		t.Fatalf("Parts error = %v, want %v", err, yacymodel.ErrInvalidHash)
	}
}
