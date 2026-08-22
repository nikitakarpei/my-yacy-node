package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	httppkg "github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
)

const testUserAgent = "test-agent (+https://example.test)"

func proxyURL(t *testing.T, handler http.HandlerFunc) (*url.URL, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return parsed, server.Close
}

func TestFetchSuccess(t *testing.T) {
	var gotUserAgent string
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>hi</html>"))
	})
	defer closeFn()

	outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/page"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if outcome.Status != pagefetch.FetchSucceeded {
		t.Fatalf("kind = %v", outcome.Status)
	}
	if string(outcome.Page.Body) != "<html>hi</html>" {
		t.Fatalf("body = %q", outcome.Page.Body)
	}
	if outcome.Page.ContentType != "text/html" {
		t.Fatalf("content type = %q", outcome.Page.ContentType)
	}
	if gotUserAgent != testUserAgent {
		t.Fatalf("user agent = %q, want %q", gotUserAgent, testUserAgent)
	}
}

func TestFetchReadsPageVersionFromSuccess(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"abc"`)
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		_, _ = w.Write([]byte("hi"))
	})
	defer closeFn()

	outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/x"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Version.EntityTag != `"abc"` {
		t.Fatalf("entity tag = %q", outcome.Version.EntityTag)
	}
	if outcome.Version.ModifiedAt.IsZero() {
		t.Fatal("modified at should be parsed")
	}
}

func TestFetchSendsConditionalHeadersWhenPageVersionKnown(t *testing.T) {
	modified := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	var gotIfNoneMatch, gotIfModifiedSince string
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, r *http.Request) {
		gotIfNoneMatch = r.Header.Get("If-None-Match")
		gotIfModifiedSince = r.Header.Get("If-Modified-Since")
		w.WriteHeader(http.StatusNotModified)
	})
	defer closeFn()

	known := pagefetch.PageVersion{EntityTag: `"abc"`, ModifiedAt: modified}
	outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/x"),
			known)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != pagefetch.FetchNotModified {
		t.Fatalf("status = %v, want FetchNotModified", outcome.Status)
	}
	if gotIfNoneMatch != `"abc"` {
		t.Fatalf("If-None-Match = %q", gotIfNoneMatch)
	}
	if gotIfModifiedSince != modified.Format(http.TimeFormat) {
		t.Fatalf("If-Modified-Since = %q", gotIfModifiedSince)
	}
}

func TestFetchOmitsConditionalHeadersWithoutAPageVersion(t *testing.T) {
	var gotIfNoneMatch, gotIfModifiedSince string
	sawHeaders := false
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, r *http.Request) {
		gotIfNoneMatch = r.Header.Get("If-None-Match")
		gotIfModifiedSince = r.Header.Get("If-Modified-Since")
		sawHeaders = true
		_, _ = w.Write([]byte("hi"))
	})
	defer closeFn()

	_, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/x"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatal(err)
	}
	if !sawHeaders {
		t.Fatal("request never reached the server")
	}
	if gotIfNoneMatch != "" || gotIfModifiedSince != "" {
		t.Fatalf("unexpected conditional headers: %q %q", gotIfNoneMatch, gotIfModifiedSince)
	}
}

func TestFetchNotModifiedKeepsTheSentPageVersion(t *testing.T) {
	modified := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	})
	defer closeFn()

	sent := pagefetch.PageVersion{EntityTag: `"abc"`, ModifiedAt: modified}
	outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/x"),
			sent)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Version.EntityTag != sent.EntityTag {
		t.Fatalf("entity tag = %q, want %q", outcome.Version.EntityTag, sent.EntityTag)
	}
	if !outcome.Version.ModifiedAt.Equal(sent.ModifiedAt) {
		t.Fatalf("modified at = %v, want %v", outcome.Version.ModifiedAt, sent.ModifiedAt)
	}
}

func TestFetchTruncatesOversizedBody(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	})
	defer closeFn()

	outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 4, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/big"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Page.Truncated || len(outcome.Page.Body) != 4 {
		t.Fatalf("expected truncation to 4 bytes, got %d truncated=%v",
			len(outcome.Page.Body), outcome.Page.Truncated)
	}
}

func TestFetchStatusMapping(t *testing.T) {
	cases := map[int]pagefetch.FetchStatus{
		http.StatusNotModified:                pagefetch.FetchNotModified,
		http.StatusTooManyRequests:            pagefetch.FetchDeferred,
		http.StatusServiceUnavailable:         pagefetch.FetchDeferred,
		http.StatusForbidden:                  pagefetch.FetchCeased,
		http.StatusUnauthorized:               pagefetch.FetchCeased,
		http.StatusUnavailableForLegalReasons: pagefetch.FetchCeased,
		http.StatusNotFound:                   pagefetch.FetchNotAPage,
		http.StatusInternalServerError:        pagefetch.FetchFailed,
	}
	for status, wantKind := range cases {
		proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		})
		outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
			Fetch(
				context.Background(),
				canonicalurltest.CanonicalURLOf(t, "http://target.example/x"),
				pagefetch.PageVersion{})
		closeFn()
		if err != nil {
			t.Fatalf("status %d: %v", status, err)
		}
		if outcome.Status != wantKind {
			t.Errorf("status %d: kind = %v, want %v", status, outcome.Status, wantKind)
		}
	}
}

func TestFetchDeferHonorsRetryAfter(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "42")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	defer closeFn()

	outcome, _ := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/x"),
			pagefetch.PageVersion{})
	if outcome.DeferFor != 42*time.Second {
		t.Fatalf("defer = %v, want 42s", outcome.DeferFor)
	}
}

func TestFetchReadsXRobotsTag(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		_, _ = w.Write([]byte("hi"))
	})
	defer closeFn()

	outcome, _ := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/x"),
			pagefetch.PageVersion{})
	if !outcome.Page.RefusesIndexing || !outcome.Page.RefusesLinkDiscovery {
		t.Fatalf("x-robots-tag not parsed: %+v", outcome)
	}
}

func TestFetchTransientOnProxyFailure(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/x"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.Status != pagefetch.FetchFailed {
		t.Fatalf("kind = %v, want transient", outcome.Status)
	}
}

func TestFetchCancelledContext(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(
			ctx,
			canonicalurltest.CanonicalURLOf(t, "http://target.example/x"),
			pagefetch.PageVersion{})
	if err == nil {
		t.Fatal("cancelled context should error")
	}
}

func TestFetchReportsTheFinalURLOfTheRedirectChain(t *testing.T) {
	for _, dialMode := range []httppkg.ProxyDialMode{
		httppkg.ProxyDialTunnel,
		httppkg.ProxyDialAbsoluteURL,
	} {
		proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/a":
				http.Redirect(w, r, "http://target.example/b", http.StatusMovedPermanently)
			case "/b":
				http.Redirect(w, r, "http://target.example/c", http.StatusFound)
			default:
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte("<html>final</html>"))
			}
		})

		outcome, err := httppkg.New(proxy, dialMode, testUserAgent, 1<<20, time.Second).
			Fetch(
				context.Background(),
				canonicalurltest.CanonicalURLOf(t, "http://target.example/a"),
				pagefetch.PageVersion{})
		closeFn()
		if err != nil {
			t.Fatalf("dial %v: Fetch: %v", dialMode, err)
		}
		if outcome.Page.FinalURL != canonicalurltest.CanonicalURLOf(t, "http://target.example/c") {
			t.Fatalf("dial %v: FinalURL = %q", dialMode, outcome.Page.FinalURL)
		}
	}
}
