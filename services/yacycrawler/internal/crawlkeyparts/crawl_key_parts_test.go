package crawlkeyparts_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlkeyparts"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

func TestAKeyOfACrawlAndAURLHashGivesBothBack(t *testing.T) {
	parts := vault.PairKey(crawlkeyparts.CrawlID, crawlkeyparts.URLHash)
	crawlID := crawlid.CrawlID{1}
	hash := urlhash.Of(canonicalurltest.CanonicalURLOf(t, "http://host/page"))

	readCrawlID, readHash, err := parts.PartsOf(parts.Key(crawlID, hash).Bytes())
	if err != nil {
		t.Fatalf("PartsOf: %v", err)
	}
	if readCrawlID != crawlID || readHash != hash {
		t.Fatalf("read %s %x, want %s %x", readCrawlID, readHash, crawlID, hash)
	}
}

func TestAKeyPartOfAnotherLengthIsRefused(t *testing.T) {
	shortKey := vault.SingleKey(vault.TextKeyPart).Key("short").Bytes()

	if _, err := vault.SingleKey(crawlkeyparts.CrawlID).PartsOf(shortKey); err == nil {
		t.Fatal("a key part of five bytes was read as a crawl ID")
	}
}
