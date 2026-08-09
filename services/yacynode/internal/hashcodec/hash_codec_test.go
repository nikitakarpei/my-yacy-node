package hashcodec_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func TestHashKeysCarryTheHashText(t *testing.T) {
	word := yacymodel.WordHash("keyword")
	layout := vaultkey.Single(hashcodec.Hash)

	if !bytes.Equal(
		layout.Key(word).Bytes(),
		vaultkey.Single(vaultkey.Text).Key(word.String()).Bytes(),
	) {
		t.Fatalf("key(%s) differs from the text key of its string form", word)
	}

	decoded, err := layout.Parts(layout.Key(word))
	if err != nil {
		t.Fatalf("Parts(%s) failed: %v", word, err)
	}
	if decoded != word {
		t.Fatalf("Parts = %s, want %s", decoded, word)
	}
}

func TestURLHashKeysCarryTheHashText(t *testing.T) {
	url, err := yacymodel.HashURL("http://example.org/page")
	if err != nil {
		t.Fatalf("HashURL failed: %v", err)
	}
	layout := vaultkey.Single(hashcodec.URLHash)

	if !bytes.Equal(
		layout.Key(url).Bytes(),
		vaultkey.Single(vaultkey.Text).Key(url.String()).Bytes(),
	) {
		t.Fatalf("key(%s) differs from the text key of its string form", url)
	}

	decoded, err := layout.Parts(layout.Key(url))
	if err != nil {
		t.Fatalf("Parts(%s) failed: %v", url, err)
	}
	if decoded != url {
		t.Fatalf("Parts = %s, want %s", decoded, url)
	}
}

func TestHashKeysFailWhenTheTextIsNoHash(t *testing.T) {
	notAHash := vaultkey.Single(vaultkey.Text).Key("too short")

	if _, err := vaultkey.Single(hashcodec.Hash).Parts(notAHash); !errors.Is(
		err,
		yacymodel.ErrInvalidHash,
	) {
		t.Fatalf("Parts error = %v, want %v", err, yacymodel.ErrInvalidHash)
	}
	if _, err := vaultkey.Single(hashcodec.URLHash).Parts(notAHash); !errors.Is(
		err,
		yacymodel.ErrInvalidHash,
	) {
		t.Fatalf("Parts error = %v, want %v", err, yacymodel.ErrInvalidHash)
	}
}
