package urlhash_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/urlhash"
)

func TestAURLHashGivesItselfBackFromItsKey(t *testing.T) {
	parts := vault.SingleKey(urlhash.KeyPart)
	hash := urlhash.Of(canonicalurltest.CanonicalURLOf(t, "http://host/page"))

	read, err := parts.PartsOf(parts.Key(hash).Bytes())
	if err != nil {
		t.Fatalf("PartsOf: %v", err)
	}
	if read != hash {
		t.Fatalf("read %x, want %x", read, hash)
	}
}

func TestAKeyPartOfAnotherLengthIsNoURLHash(t *testing.T) {
	shortKey := vault.SingleKey(vault.TextKeyPart).Key("short").Bytes()

	if _, err := vault.SingleKey(urlhash.KeyPart).PartsOf(shortKey); err == nil {
		t.Fatal("a key part of five bytes was read as a URL hash")
	}
}
