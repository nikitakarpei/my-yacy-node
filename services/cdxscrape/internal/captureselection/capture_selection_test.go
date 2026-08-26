package captureselection_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/captureselection"
	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/cdxindex"
)

func TestNewestCapturesOfKeepsOneCapturePerPage(t *testing.T) {
	newestCaptures := captureselection.NewestCapturesOf([]cdxindex.Capture{
		{URLKey: "com,example)/", Timestamp: "20240101120000", OriginalURL: "https://example.com/"},
		{URLKey: "com,example)/", Timestamp: "20240501120000", OriginalURL: "https://example.com/"},
		{URLKey: "com,example)/", Timestamp: "20240301120000", OriginalURL: "https://example.com/"},
	})

	if len(newestCaptures) != 1 {
		t.Fatalf("captures = %v, want one", newestCaptures)
	}
	if newestCaptures[0].Timestamp != "20240501120000" {
		t.Fatalf("timestamp = %q, want the newest", newestCaptures[0].Timestamp)
	}
}

func TestNewestCapturesOfKeepsEveryPageInTheOrderTheArchiveListedThem(t *testing.T) {
	newestCaptures := captureselection.NewestCapturesOf([]cdxindex.Capture{
		{URLKey: "com,example)/", Timestamp: "20240101120000"},
		{URLKey: "com,example)/about", Timestamp: "20240101120000"},
		{URLKey: "com,example)/", Timestamp: "20240501120000"},
		{URLKey: "com,example)/contact", Timestamp: "20240101120000"},
	})

	wantedKeys := []string{"com,example)/", "com,example)/about", "com,example)/contact"}
	if len(newestCaptures) != len(wantedKeys) {
		t.Fatalf("captures = %v, want %v", newestCaptures, wantedKeys)
	}
	for at, capture := range newestCaptures {
		if capture.URLKey != wantedKeys[at] {
			t.Fatalf("capture %d = %q, want %q", at, capture.URLKey, wantedKeys[at])
		}
	}
}

func TestNewestCapturesOfKeepsNothingFromAnEmptyArchiveAnswer(t *testing.T) {
	if newestCaptures := captureselection.NewestCapturesOf(nil); len(newestCaptures) != 0 {
		t.Fatalf("captures = %v, want none", newestCaptures)
	}
}
