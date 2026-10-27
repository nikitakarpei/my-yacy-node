package pagecacheproxy_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/cachedpages"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/pagecacheproxy"
)

const (
	maxBodyBytesInTest = 64
	stopDeadline       = 10 * time.Second
	userAgent          = "yacypeer (+https://yacy.net)"
	pageBody           = "<p>page</p>"
)

type stoppedClock struct{}

func (stoppedClock) Now() time.Time {
	return time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
}

func (stoppedClock) Sleep(ctx context.Context, _ time.Duration) error {
	<-ctx.Done()
	return fmt.Errorf("sleep: %w", ctx.Err())
}

type origin struct {
	answers  map[string]http.HandlerFunc
	requests map[string]int
	mu       sync.Mutex
}

func (o *origin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o.mu.Lock()
	o.requests[r.URL.Path]++
	o.mu.Unlock()
	o.answers[r.URL.Path](w, r)
}

func (o *origin) amountOfRequestsFor(path string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.requests[path]
}

func answerWith(statusCode int, body string, header http.Header) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		for name, values := range header {
			w.Header()[name] = values
		}
		w.WriteHeader(statusCode)
		_, _ = io.WriteString(w, body)
	}
}

func originAnswering(t *testing.T, answers map[string]http.HandlerFunc) (*origin, *url.URL) {
	t.Helper()
	pages := &origin{answers: answers, requests: map[string]int{}}
	server := httptest.NewServer(pages)
	t.Cleanup(server.Close)
	return pages, urlOf(t, server.URL)
}

func urlOf(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %q: %v", rawURL, err)
	}
	return parsed
}

func runningProxy(t *testing.T, egressProxyURL *url.URL) *pagecacheproxy.PageCacheProxy {
	t.Helper()
	cachedPages, err := cachedpages.Open(t.Context(), t.TempDir(), time.Hour, 1<<20, stoppedClock{})
	if err != nil {
		t.Fatalf("open page cachedPages: %v", err)
	}
	proxy, err := pagecacheproxy.Listen(
		t.Context(),
		cachedPages,
		egressProxyURL,
		maxBodyBytesInTest,
	)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- proxy.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-runDone:
			if err != nil {
				t.Errorf("Run: %v", err)
			}
		case <-time.After(stopDeadline):
			t.Error("Run did not stop after cancel")
		}
	})
	return proxy
}

type fetchedPage struct {
	header     http.Header
	body       string
	statusCode int
}

func fetchThrough(
	t *testing.T,
	proxy *pagecacheproxy.PageCacheProxy,
	pageURL string,
	header http.Header,
) fetchedPage {
	t.Helper()
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(urlOf(t, proxy.URL()))},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, pageURL, nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	request.Header = header.Clone()
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("fetch through the proxy: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read the answer: %v", err)
	}
	return fetchedPage{header: response.Header, body: string(body), statusCode: response.StatusCode}
}

func plainFetch() http.Header {
	return http.Header{"User-Agent": {userAgent}}
}

func assertOriginRequests(t *testing.T, pages *origin, path string, want int) {
	t.Helper()
	if got := pages.amountOfRequestsFor(path); got != want {
		t.Fatalf("origin got %d requests for %s, want %d", got, path, want)
	}
}

func TestASecondFetchOfAPageIsAnsweredFromTheCache(t *testing.T) {
	pages, egressURL := originAnswering(t, map[string]http.HandlerFunc{
		"/page": answerWith(http.StatusOK, pageBody, http.Header{"X-Robots-Tag": {"noarchive"}}),
	})
	proxy := runningProxy(t, egressURL)

	first := fetchThrough(t, proxy, "http://pages.test/page", plainFetch())
	second := fetchThrough(t, proxy, "http://pages.test/page", plainFetch())

	assertOriginRequests(t, pages, "/page", 1)
	for _, fetched := range []fetchedPage{first, second} {
		if fetched.statusCode != http.StatusOK || fetched.body != pageBody ||
			fetched.header.Get("X-Robots-Tag") != "noarchive" {
			t.Fatalf("fetched %+v, want the page with its robots tag", fetched)
		}
	}
}

func TestAFetchWithAnotherUserAgentGoesToTheOrigin(t *testing.T) {
	pages, egressURL := originAnswering(t, map[string]http.HandlerFunc{
		"/page": answerWith(http.StatusOK, pageBody, nil),
	})
	proxy := runningProxy(t, egressURL)

	fetchThrough(t, proxy, "http://pages.test/page", plainFetch())
	fetchThrough(t, proxy, "http://pages.test/page", http.Header{"User-Agent": {"another"}})

	assertOriginRequests(t, pages, "/page", 2)
}

func TestAConditionalFetchGoesToTheOriginAndCachesNothing(t *testing.T) {
	pages, egressURL := originAnswering(t, map[string]http.HandlerFunc{
		"/page": answerWith(http.StatusOK, pageBody, nil),
	})
	proxy := runningProxy(t, egressURL)
	conditionalFetch := plainFetch()
	conditionalFetch.Set("If-None-Match", `"v1"`)

	fetchThrough(t, proxy, "http://pages.test/page", conditionalFetch)
	fetchThrough(t, proxy, "http://pages.test/page", plainFetch())

	assertOriginRequests(t, pages, "/page", 2)
}

func TestAnAnswerToWaitIsRelayedAndCachesNothing(t *testing.T) {
	pages, egressURL := originAnswering(t, map[string]http.HandlerFunc{
		"/busy": answerWith(http.StatusServiceUnavailable, "", http.Header{"Retry-After": {"60"}}),
	})
	proxy := runningProxy(t, egressURL)

	fetchThrough(t, proxy, "http://pages.test/busy", plainFetch())
	fetched := fetchThrough(t, proxy, "http://pages.test/busy", plainFetch())

	assertOriginRequests(t, pages, "/busy", 2)
	if fetched.statusCode != http.StatusServiceUnavailable ||
		fetched.header.Get("Retry-After") != "60" {
		t.Fatalf("fetched %+v, want the origin's answer to wait", fetched)
	}
}

func TestFinalAnswersAreKept(t *testing.T) {
	for name, answer := range map[string]http.HandlerFunc{
		"missing":    answerWith(http.StatusNotFound, "", nil),
		"gone":       answerWith(http.StatusGone, "", nil),
		"redirected": answerWith(http.StatusMovedPermanently, "", http.Header{"Location": {"/new"}}),
	} {
		t.Run(name, func(t *testing.T) {
			pages, egressURL := originAnswering(t, map[string]http.HandlerFunc{"/page": answer})
			proxy := runningProxy(t, egressURL)

			first := fetchThrough(t, proxy, "http://pages.test/page", plainFetch())
			second := fetchThrough(t, proxy, "http://pages.test/page", plainFetch())

			assertOriginRequests(t, pages, "/page", 1)
			if second.statusCode != first.statusCode ||
				second.header.Get("Location") != first.header.Get("Location") {
				t.Fatalf("cache answered %+v, origin answered %+v", second, first)
			}
		})
	}
}

func TestARedirectWithoutATargetCachesNothing(t *testing.T) {
	pages, egressURL := originAnswering(t, map[string]http.HandlerFunc{
		"/page": answerWith(http.StatusFound, "", nil),
	})
	proxy := runningProxy(t, egressURL)

	fetchThrough(t, proxy, "http://pages.test/page", plainFetch())
	fetchThrough(t, proxy, "http://pages.test/page", plainFetch())

	assertOriginRequests(t, pages, "/page", 2)
}

func TestAPageOverTheMaximumIsRelayedWholeAndCachesNothing(t *testing.T) {
	largeBody := strings.Repeat("x", 3*maxBodyBytesInTest)
	pages, egressURL := originAnswering(t, map[string]http.HandlerFunc{
		"/large": answerWith(http.StatusOK, largeBody, nil),
	})
	proxy := runningProxy(t, egressURL)

	first := fetchThrough(t, proxy, "http://pages.test/large", plainFetch())
	fetchThrough(t, proxy, "http://pages.test/large", plainFetch())

	assertOriginRequests(t, pages, "/large", 2)
	if first.body != largeBody {
		t.Fatalf("fetched %d bytes, want the whole %d", len(first.body), len(largeBody))
	}
}

func TestCookiesOfTheOriginAreNotRelayed(t *testing.T) {
	_, egressURL := originAnswering(t, map[string]http.HandlerFunc{
		"/page": answerWith(http.StatusOK, pageBody, http.Header{"Set-Cookie": {"session=1"}}),
	})
	proxy := runningProxy(t, egressURL)

	fetched := fetchThrough(t, proxy, "http://pages.test/page", plainFetch())

	if cookie := fetched.header.Get("Set-Cookie"); cookie != "" {
		t.Fatalf("relayed Set-Cookie %q, want none", cookie)
	}
}

func TestAnUnreachableOriginAnswersBadGateway(t *testing.T) {
	closedServer := httptest.NewServer(http.NotFoundHandler())
	closedServer.Close()
	proxy := runningProxy(t, urlOf(t, closedServer.URL))

	fetched := fetchThrough(t, proxy, "http://pages.test/page", plainFetch())

	if fetched.statusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", fetched.statusCode, http.StatusBadGateway)
	}
}

func TestTheProxyRefusesARequestWithoutAnAbsoluteURL(t *testing.T) {
	_, egressURL := originAnswering(t, map[string]http.HandlerFunc{})
	proxy := runningProxy(t, egressURL)
	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		proxy.URL()+"/page",
		nil,
	)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request the proxy: %v", err)
	}
	_ = response.Body.Close()

	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestCloseReleasesTheListener(t *testing.T) {
	cachedPages, err := cachedpages.Open(t.Context(), t.TempDir(), time.Hour, 1<<20, stoppedClock{})
	if err != nil {
		t.Fatalf("open page cachedPages: %v", err)
	}
	proxy, err := pagecacheproxy.Listen(
		t.Context(),
		cachedPages,
		urlOf(t, "http://127.0.0.1:1"),
		maxBodyBytesInTest,
	)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	if err := proxy.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := proxy.Close(); err == nil {
		t.Fatal("a second Close succeeded, want the listener already released")
	}
}
