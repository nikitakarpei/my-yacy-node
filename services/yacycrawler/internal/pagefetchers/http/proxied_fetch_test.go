package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	httppkg "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchers/http"
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
		Fetch(context.Background(), "http://target.example/page")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if outcome.Status != pagevisit.FetchSucceeded {
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

func TestFetchTruncatesOversizedBody(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	})
	defer closeFn()

	outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 4, time.Second).
		Fetch(context.Background(), "http://target.example/big")
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Page.Truncated || len(outcome.Page.Body) != 4 {
		t.Fatalf("expected truncation to 4 bytes, got %d truncated=%v",
			len(outcome.Page.Body), outcome.Page.Truncated)
	}
}

func TestFetchStatusMapping(t *testing.T) {
	cases := map[int]pagevisit.FetchStatus{
		http.StatusTooManyRequests:            pagevisit.FetchDeferred,
		http.StatusServiceUnavailable:         pagevisit.FetchDeferred,
		http.StatusForbidden:                  pagevisit.FetchCeased,
		http.StatusUnauthorized:               pagevisit.FetchCeased,
		http.StatusUnavailableForLegalReasons: pagevisit.FetchCeased,
		http.StatusNotFound:                   pagevisit.FetchNotAPage,
		http.StatusInternalServerError:        pagevisit.FetchFailed,
	}
	for status, wantKind := range cases {
		proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		})
		outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
			Fetch(context.Background(), "http://target.example/x")
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
		Fetch(context.Background(), "http://target.example/x")
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
		Fetch(context.Background(), "http://target.example/x")
	if !outcome.Page.RefusesIndexing || !outcome.Page.RefusesLinkDiscovery {
		t.Fatalf("x-robots-tag not parsed: %+v", outcome)
	}
}

func TestFetchTransientOnProxyFailure(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(context.Background(), "http://target.example/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.Status != pagevisit.FetchFailed {
		t.Fatalf("kind = %v, want transient", outcome.Status)
	}
}

func TestFetchCancelledContext(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(ctx, "http://target.example/x")
	if err == nil {
		t.Fatal("cancelled context should error")
	}
}

func TestFetchRecordsRedirectChain(t *testing.T) {
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
			Fetch(context.Background(), "http://target.example/a")
		closeFn()
		if err != nil {
			t.Fatalf("dial %v: Fetch: %v", dialMode, err)
		}
		if outcome.Page.FinalURL != "http://target.example/c" {
			t.Fatalf("dial %v: FinalURL = %q", dialMode, outcome.Page.FinalURL)
		}
		want := []string{
			"http://target.example/a",
			"http://target.example/b",
			"http://target.example/c",
		}
		if len(outcome.Page.RedirectChain) != len(want) {
			t.Fatalf("dial %v: chain = %v, want %v", dialMode, outcome.Page.RedirectChain, want)
		}
		for i := range want {
			if outcome.Page.RedirectChain[i] != want[i] {
				t.Fatalf(
					"dial %v: chain[%d] = %q, want %q",
					dialMode,
					i,
					outcome.Page.RedirectChain[i],
					want[i],
				)
			}
		}
	}
}

func TestFetchDirectRecordsSingleHopChain(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>hi</html>"))
	})
	defer closeFn()

	outcome, err := httppkg.New(proxy, httppkg.ProxyDialTunnel, testUserAgent, 1<<20, time.Second).
		Fetch(context.Background(), "http://target.example/page")
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Page.RedirectChain) != 1 ||
		outcome.Page.RedirectChain[0] != "http://target.example/page" {
		t.Fatalf("chain = %v", outcome.Page.RedirectChain)
	}
}
