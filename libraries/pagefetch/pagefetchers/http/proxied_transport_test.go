package http_test

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	httppkg "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
)

func TestFetchAbsoluteURLModeSendsNoConnect(t *testing.T) {
	var gotMethod, gotRequestURI string
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotRequestURI = r.RequestURI
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>hi</html>"))
	})
	defer closeFn()

	outcome, err := httppkg.New(httppkg.Proxy{URL: proxy, DialMode: httppkg.ProxyDialAbsoluteURL, Dial: (&net.Dialer{}).DialContext}, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "https://target.example/page"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if outcome.Status != pagefetch.FetchSucceeded {
		t.Fatalf("kind = %v", outcome.Status)
	}
	if gotMethod == http.MethodConnect {
		t.Fatal("absolute-url mode must not send CONNECT")
	}
	if gotRequestURI != "https://target.example/page" {
		t.Fatalf("request-uri = %q, want absolute URI", gotRequestURI)
	}
}

func TestFetchAbsoluteURLModeTransientOnDialFailure(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	outcome, err := httppkg.New(httppkg.Proxy{URL: proxy, DialMode: httppkg.ProxyDialAbsoluteURL, Dial: (&net.Dialer{}).DialContext}, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "https://target.example/x"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.Status != pagefetch.FetchFailed {
		t.Fatalf("kind = %v, want transient", outcome.Status)
	}
}

func TestFetchAbsoluteURLModeHandlesHTTPTarget(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("plain"))
	})
	defer closeFn()

	outcome, err := httppkg.New(httppkg.Proxy{URL: proxy, DialMode: httppkg.ProxyDialAbsoluteURL, Dial: (&net.Dialer{}).DialContext}, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://target.example/page"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(outcome.Page.Body) != "plain" {
		t.Fatalf("body = %q", outcome.Page.Body)
	}
}

func TestFetchAbsoluteURLModeGivesUpWhenTheDeadlinePasses(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)
		_, _ = w.Write([]byte("far too late"))
	})
	defer closeFn()

	startedAt := time.Now()
	outcome, err := httppkg.New(httppkg.Proxy{URL: proxy, DialMode: httppkg.ProxyDialAbsoluteURL, Dial: (&net.Dialer{}).DialContext}, testUserAgent, 1<<20, 200*time.Millisecond).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "https://target.example/slow"),
			pagefetch.PageVersion{})
	spent := time.Since(startedAt)

	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if outcome.Status != pagefetch.FetchDeadlinePassed {
		t.Fatalf("status = %v, want the fetch to give up on its deadline", outcome.Status)
	}
	if spent > time.Second {
		t.Fatalf("Fetch took %v, want it to give up near its 200ms deadline", spent)
	}
}

func TestFetchAbsoluteURLModeGivesUpWhenTheReaderStopsWaiting(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)
		_, _ = w.Write([]byte("far too late"))
	})
	defer closeFn()

	ctx, stopWaiting := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		stopWaiting()
	}()

	startedAt := time.Now()
	_, err := httppkg.New(httppkg.Proxy{URL: proxy, DialMode: httppkg.ProxyDialAbsoluteURL, Dial: (&net.Dialer{}).DialContext}, testUserAgent, 1<<20, time.Minute).
		Fetch(ctx, canonicalurltest.CanonicalURLOf(t, "https://target.example/slow"), pagefetch.PageVersion{})
	spent := time.Since(startedAt)

	if err == nil {
		t.Fatal("Fetch kept reading after the reader stopped waiting")
	}
	if spent > time.Second {
		t.Fatalf("Fetch took %v, want it to stop when the context was cancelled", spent)
	}
}

func TestFetchAbsoluteURLModeSendsTheProxyCredentials(t *testing.T) {
	var gotAuthorization string
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Proxy-Authorization")
		_, _ = w.Write([]byte("plain"))
	})
	defer closeFn()
	proxy.User = url.UserPassword("", "s3cret")

	_, err := httppkg.New(httppkg.Proxy{URL: proxy, DialMode: httppkg.ProxyDialAbsoluteURL, Dial: (&net.Dialer{}).DialContext}, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "https://target.example/page"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if gotAuthorization != "Basic OnMzY3JldA==" {
		t.Fatalf(
			"Proxy-Authorization = %q, want the credentials of the proxy URL",
			gotAuthorization,
		)
	}
}

func TestFetchAbsoluteURLModeSendsNoProxyCredentialsWhenTheProxyURLHasNone(t *testing.T) {
	var gotAuthorization []string
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Values("Proxy-Authorization")
		_, _ = w.Write([]byte("plain"))
	})
	defer closeFn()

	_, err := httppkg.New(httppkg.Proxy{URL: proxy, DialMode: httppkg.ProxyDialAbsoluteURL, Dial: (&net.Dialer{}).DialContext}, testUserAgent, 1<<20, time.Second).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "https://target.example/page"),
			pagefetch.PageVersion{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(gotAuthorization) != 0 {
		t.Fatalf("Proxy-Authorization = %q, want none", gotAuthorization)
	}
}

func TestFetchDialsTheProxyThroughTheGivenDial(t *testing.T) {
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>hi</html>"))
	})
	defer closeFn()
	namedProxy := &url.URL{Scheme: "http", Host: "proxy.internal:80"}
	dialProxy := func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, proxy.Host)
	}

	for _, dialMode := range []httppkg.ProxyDialMode{httppkg.ProxyDialTunnel, httppkg.ProxyDialAbsoluteURL} {
		outcome, err := httppkg.New(httppkg.Proxy{URL: namedProxy, DialMode: dialMode, Dial: dialProxy}, testUserAgent, 1<<20, time.Second).
			Fetch(
				context.Background(),
				canonicalurltest.CanonicalURLOf(t, "http://target.example/page"),
				pagefetch.PageVersion{})
		if err != nil {
			t.Fatalf("Fetch in mode %v: %v", dialMode, err)
		}
		if outcome.Status != pagefetch.FetchSucceeded {
			t.Errorf(
				"status in mode %v = %v, want %v",
				dialMode,
				outcome.Status,
				pagefetch.FetchSucceeded,
			)
		}
	}
}
