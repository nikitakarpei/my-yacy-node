package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	httppkg "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
)

const (
	replayURL   = "http://pywb.example/capture/20260824134801id_/https://example.com/"
	originalURL = "https://example.com/"

	replayLinkHeader = `<https://example.com/>; rel="original", ` +
		`<http://pywb.example/capture/https://example.com/>; rel="timegate", ` +
		`<http://pywb.example/capture/timemap/link/https://example.com/>; ` +
		`rel="timemap"; type="application/link-format", ` +
		`<http://pywb.example/capture/20260824134801mp_/https://example.com/>; ` +
		`rel="memento"; datetime="Mon, 24 Aug 2026 13:48:01 GMT"; collection="capture"`

	capturedAt       = "Mon, 24 Aug 2026 13:48:01 GMT"
	originModifiedAt = "Tue, 18 Aug 2026 20:06:42 GMT"
)

func fetchReplay(t *testing.T, headers map[string]string) pagefetch.FetchOutcome {
	t.Helper()
	return fetchReplayThrough(t, headers, func(proxied http.RoundTripper) http.RoundTripper {
		return httppkg.NewReplayedCaptureTransport(proxied)
	})
}

func fetchReplayThrough(
	t *testing.T,
	headers map[string]string,
	transportAround func(proxied http.RoundTripper) http.RoundTripper,
) pagefetch.FetchOutcome {
	t.Helper()
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		for name, value := range headers {
			w.Header().Set(name, value)
		}
		_, _ = w.Write([]byte("<html>archived</html>"))
	})
	defer closeFn()

	transport := transportAround(httppkg.NewProxiedTransport(proxy, httppkg.ProxyDialTunnel))
	outcome, err := httppkg.New(transport, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, replayURL),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if outcome.Status != pagefetch.FetchSucceeded {
		t.Fatalf("status = %v", outcome.Status)
	}
	return outcome
}

func TestFetchReadsReplayedPageAsItsOriginal(t *testing.T) {
	outcome := fetchReplay(t, map[string]string{
		"Memento-Datetime":             capturedAt,
		"Link":                         replayLinkHeader,
		"ETag":                         `"replay-tag"`,
		"X-Archive-Orig-ETag":          `"origin-tag"`,
		"X-Archive-Orig-Last-Modified": originModifiedAt,
		"X-Archive-Orig-Date":          capturedAt,
	})

	if outcome.Page.FinalURL.String() != originalURL {
		t.Fatalf("final url = %q, want %q", outcome.Page.FinalURL, originalURL)
	}
	if outcome.Version.EntityTag != `"origin-tag"` {
		t.Fatalf("entity tag = %q", outcome.Version.EntityTag)
	}
	wanted, err := http.ParseTime(originModifiedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Version.ModifiedAt.Equal(wanted) {
		t.Fatalf("modified at = %v, want %v", outcome.Version.ModifiedAt, wanted)
	}
}

func TestFetchReadsReplayedPageWhenOriginalIsNamedLast(t *testing.T) {
	outcome := fetchReplay(t, map[string]string{
		"Memento-Datetime": capturedAt,
		"Link": `<http://pywb.example/capture/20260824134801mp_/https://example.com/>; ` +
			`rel="memento"; datetime="Mon, 24 Aug 2026 13:48:01 GMT", ` +
			`<https://example.com/>; rel="original"`,
	})

	if outcome.Page.FinalURL.String() != originalURL {
		t.Fatalf("final url = %q, want %q", outcome.Page.FinalURL, originalURL)
	}
}

func TestFetchKeepsReplayURLWhenNoReplayedCaptureTransportIsComposed(t *testing.T) {
	outcome := fetchReplayThrough(t, map[string]string{
		"Memento-Datetime":             capturedAt,
		"Link":                         replayLinkHeader,
		"ETag":                         `"replay-tag"`,
		"X-Archive-Orig-ETag":          `"origin-tag"`,
		"X-Archive-Orig-Last-Modified": originModifiedAt,
	}, func(proxied http.RoundTripper) http.RoundTripper { return proxied })

	wanted := canonicalurltest.CanonicalURLOf(t, replayURL)
	if outcome.Page.FinalURL.String() != wanted.String() {
		t.Fatalf("final url = %q, want %q", outcome.Page.FinalURL, wanted)
	}
	if outcome.Version.EntityTag != `"replay-tag"` {
		t.Fatalf("entity tag = %q, want the replay's own", outcome.Version.EntityTag)
	}
}

func TestFetchKeepsReplayURLWhenNoOriginalIsNamed(t *testing.T) {
	outcome := fetchReplay(t, map[string]string{
		"Memento-Datetime": capturedAt,
		"Link":             `<http://pywb.example/capture/https://example.com/>; rel="timegate"`,
	})

	wanted := canonicalurltest.CanonicalURLOf(t, replayURL)
	if outcome.Page.FinalURL.String() != wanted.String() {
		t.Fatalf("final url = %q, want %q", outcome.Page.FinalURL, wanted)
	}
}
