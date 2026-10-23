package crawlid_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

func mintedCrawlID(t *testing.T) crawlid.CrawlID {
	t.Helper()
	id, err := crawlid.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return id
}

func TestACrawlIDIsAVersionSevenUUID(t *testing.T) {
	if text := mintedCrawlID(t).String(); len(text) != 36 || text[14] != '7' {
		t.Fatalf("crawl ID %s is not a version 7 UUID", text)
	}
}

func TestALaterCrawlIDSortsAfterAnEarlierOne(t *testing.T) {
	earlier := mintedCrawlID(t)
	later := mintedCrawlID(t)

	if earlier.String() >= later.String() {
		t.Fatalf("crawl ID %s does not sort before %s", earlier, later)
	}
}

func TestACrawlIDSurvivesItsTextForm(t *testing.T) {
	id := mintedCrawlID(t)

	var read crawlid.CrawlID
	if err := read.UnmarshalText([]byte(id.String())); err != nil {
		t.Fatalf("UnmarshalText(%s): %v", id, err)
	}
	if read != id {
		t.Fatalf("read %s, want %s", read, id)
	}
}

func TestATextThatIsNoCrawlIDIsRefused(t *testing.T) {
	for _, text := range []string{"", "0190a1b2-c3d4-7e5f-8a6b-7c8d9e0f1a2z"} {
		var read crawlid.CrawlID
		if err := read.UnmarshalText([]byte(text)); err == nil {
			t.Errorf("UnmarshalText(%q) read %s, want an error", text, read)
		}
	}
}
