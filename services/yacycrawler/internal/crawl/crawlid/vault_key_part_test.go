package crawlid_test

import (
	"bytes"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

func TestACrawlIDGivesItselfBackFromItsKey(t *testing.T) {
	for _, part := range []vault.KeyPart[crawlid.CrawlID]{crawlid.KeyPart, crawlid.KeyPartNewestFirst} {
		parts := vault.SingleKey(part)
		id := mintedCrawlID(t)

		read, err := parts.PartsOf(parts.Key(id).Bytes())
		if err != nil {
			t.Fatalf("PartsOf: %v", err)
		}
		if read != id {
			t.Fatalf("read %s, want %s", read, id)
		}
	}
}

func TestAKeyOfALaterCrawlSortsAfterAnEarlierOne(t *testing.T) {
	parts := vault.SingleKey(crawlid.KeyPart)
	earlier := mintedCrawlID(t)
	later := mintedCrawlID(t)

	if bytes.Compare(parts.Key(earlier).Bytes(), parts.Key(later).Bytes()) >= 0 {
		t.Fatalf("the key of %s does not sort before the key of %s", earlier, later)
	}
}

func TestANewestFirstKeyOfALaterCrawlSortsBeforeAnEarlierOne(t *testing.T) {
	parts := vault.SingleKey(crawlid.KeyPartNewestFirst)
	earlier := mintedCrawlID(t)
	later := mintedCrawlID(t)

	if bytes.Compare(parts.Key(later).Bytes(), parts.Key(earlier).Bytes()) >= 0 {
		t.Fatalf("the newest first key of %s does not sort before the key of %s", later, earlier)
	}
}

func TestAKeyPartOfAnotherLengthIsNoCrawlID(t *testing.T) {
	shortKey := vault.SingleKey(vault.TextKeyPart).Key("short").Bytes()

	for _, part := range []vault.KeyPart[crawlid.CrawlID]{crawlid.KeyPart, crawlid.KeyPartNewestFirst} {
		if _, err := vault.SingleKey(part).PartsOf(shortKey); err == nil {
			t.Fatal("a key part of five bytes was read as a crawl ID")
		}
	}
}
