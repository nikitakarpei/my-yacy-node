package capturereplay_test

import (
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/capturereplay"
	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/cdxindex"
)

func TestReplayURLOfCarriesTheCollectionTheMomentAndTheCapturedURL(t *testing.T) {
	archive := archiveAt(t, "http://pywb:8080")

	replayURL := replayURLOf(t, archive, cdxindex.Capture{
		Timestamp:   "20240101120000",
		OriginalURL: "https://example.com/a/page",
	})

	wanted := "http://pywb:8080/archive/20240101120000mp_/" +
		"https:%2F%2Fexample.com%2Fa%2Fpage"
	if replayURL != wanted {
		t.Fatalf("replay url = %q, want %q", replayURL, wanted)
	}
}

func TestReplayURLOfKeepsTheCapturedURLWholeThroughCanonicalForm(t *testing.T) {
	archive := archiveAt(t, "http://pywb:8080")

	for _, capturedURL := range []string{
		"https://example.com/a/page",
		"https://example.com/a/../page",
		"https://example.com/a/page?query=1&other=2",
		"https://example.com/",
	} {
		replayURL := replayURLOf(t, archive, cdxindex.Capture{
			Timestamp:   "20240101120000",
			OriginalURL: capturedURL,
		})

		readBack, err := url.PathUnescape(
			replayURL[len("http://pywb:8080/archive/20240101120000mp_/"):],
		)
		if err != nil {
			t.Fatalf("unescape %q: %v", replayURL, err)
		}
		if readBack != capturedURL {
			t.Errorf("replay url carries %q, want %q", readBack, capturedURL)
		}
	}
}

func TestReplayURLOfKeepsATrailingArchivePathElement(t *testing.T) {
	archive := archiveAt(t, "http://pywb:8080/wayback/")

	replayURL := replayURLOf(t, archive, cdxindex.Capture{
		Timestamp:   "20240101120000",
		OriginalURL: "https://example.com/",
	})

	wanted := "http://pywb:8080/wayback/archive/20240101120000mp_/https:%2F%2Fexample.com%2F"
	if replayURL != wanted {
		t.Fatalf("replay url = %q, want %q", replayURL, wanted)
	}
}

func TestReplayURLOfFailsWhenTheArchiveAddressHasNoHost(t *testing.T) {
	archive := archiveAt(t, "http:///")

	if _, err := archive.ReplayURLOf(cdxindex.Capture{
		Timestamp:   "20240101120000",
		OriginalURL: "https://example.com/",
	}); err == nil {
		t.Fatal("replay url of capture: want an error")
	}
}

func replayURLOf(
	t *testing.T,
	archive *capturereplay.Archive,
	capture cdxindex.Capture,
) string {
	t.Helper()
	replayURL, err := archive.ReplayURLOf(capture)
	if err != nil {
		t.Fatalf("replay url of capture %v: %v", capture, err)
	}
	return replayURL.String()
}

func archiveAt(t *testing.T, archiveURL string) *capturereplay.Archive {
	t.Helper()
	parsed, err := url.Parse(archiveURL)
	if err != nil {
		t.Fatalf("parse archive url %q: %v", archiveURL, err)
	}
	return capturereplay.New(parsed, "archive")
}
