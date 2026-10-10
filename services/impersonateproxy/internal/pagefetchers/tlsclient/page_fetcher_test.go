package tlsclient_test

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/chromenavigation"
	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/egressproxytest"
	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/pagefetchers/tlsclient"
)

const pageBody = "<html><title>Page</title></html>"

type fetchRecord struct {
	lock      sync.Mutex
	statuses  []int
	failures  []error
	cancelled []string
}

func (r *fetchRecord) PageFetched(_ context.Context, _ string, status int) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.statuses = append(r.statuses, status)
}

func (r *fetchRecord) FetchFailed(_ context.Context, _ string, cause error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.failures = append(r.failures, cause)
}

func (r *fetchRecord) FetchCancelled(_ context.Context, address string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cancelled = append(r.cancelled, address)
}

func TestASecurePageIsFetchedAsChromeOverHTTP2ThroughATunnel(t *testing.T) {
	var originRequests []*http.Request
	origin := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			originRequests = append(originRequests, r)
			_, _ = io.WriteString(w, pageBody)
		},
	))
	origin.EnableHTTP2 = true
	origin.StartTLS()
	defer origin.Close()
	egressProxy := egressproxytest.Start(t)
	fetches := &fetchRecord{}
	pageFetcher := pageFetcherFor(t, egressProxy, trustedRootsOf(origin), fetches)

	status, headers, body := fetchedPageOf(t, pageFetcher, http.MethodGet, origin.URL+"/page")

	if status != http.StatusOK || body != pageBody || headers.Get("Content-Length") == "" {
		t.Fatalf("status %d, headers %v, body %q", status, headers, body)
	}
	originRequest := originRequests[0]
	if originRequest.ProtoMajor != 2 ||
		!strings.Contains(originRequest.UserAgent(), "Chrome/152.0.0.0") ||
		originRequest.Header.Get("Sec-Fetch-Mode") != "navigate" {
		t.Fatalf("origin request %s %v", originRequest.Proto, originRequest.Header)
	}
	if !slices.Equal(egressProxy.TunnelTargets(), []string{origin.Listener.Addr().String()}) ||
		!slices.Equal(fetches.statuses, []int{http.StatusOK}) {
		t.Fatalf("tunnel targets %v, fetched statuses %v",
			egressProxy.TunnelTargets(), fetches.statuses)
	}
}

func TestARedirectIsReturnedAndNotFollowed(t *testing.T) {
	originRequests := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originRequests++
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	}))
	defer origin.Close()
	pageFetcher := pageFetcherFor(t, egressproxytest.Start(t), nil, &fetchRecord{})

	status, headers, _ := fetchedPageOf(t, pageFetcher, http.MethodGet, origin.URL+"/page")

	if status != http.StatusFound || headers.Get("Location") != "/elsewhere" ||
		originRequests != 1 {
		t.Fatalf("status %d, headers %v, origin requests %d", status, headers, originRequests)
	}
}

func TestACompressedPageIsDecoded(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Length", "1000")
		if r.Method == http.MethodHead {
			return
		}
		w.Header().Del("Content-Length")
		writer := gzip.NewWriter(w)
		_, _ = io.WriteString(writer, pageBody)
		_ = writer.Close()
	}))
	defer origin.Close()
	pageFetcher := pageFetcherFor(t, egressproxytest.Start(t), nil, &fetchRecord{})

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		status, headers, body := fetchedPageOf(t, pageFetcher, method, origin.URL+"/page")

		wantBody := map[string]string{http.MethodGet: pageBody, http.MethodHead: ""}[method]
		if status != http.StatusOK || body != wantBody ||
			headers.Get("Content-Encoding") != "" || headers.Get("Content-Length") != "" {
			t.Fatalf("%s: status %d, headers %v, body %q", method, status, headers, body)
		}
	}
}

func TestAnInsecurePageIsAskedWithTheHeaderOrderOfChrome(t *testing.T) {
	origin, headerNames := rawOrigin(t)
	pageFetcher := pageFetcherFor(t, egressproxytest.Start(t), nil, &fetchRecord{})

	status, _, _ := fetchedPageOf(t, pageFetcher, http.MethodGet, "http://"+origin+"/page")

	want := []string{
		"Host", "Connection", "Upgrade-Insecure-Requests", "User-Agent", "Accept",
		"Accept-Encoding", "Accept-Language",
	}
	if received := <-headerNames; status != http.StatusOK || !slices.Equal(received, want) {
		t.Fatalf("status %d, header names %v", status, received)
	}
}

func TestARefusedTunnelIsReportedAsAFailedFetch(t *testing.T) {
	fetches := &fetchRecord{}
	pageFetcher := pageFetcherFor(t, egressproxytest.StartRefusing(t), nil, fetches)

	wasFetched := wasFetchedBy(t.Context(), pageFetcher, requestFor(t, http.MethodGet,
		"https://site.example/page"))

	if wasFetched || len(fetches.failures) != 1 || len(fetches.cancelled) != 0 {
		t.Fatalf("fetched %v, failures %v, cancelled %v",
			wasFetched, fetches.failures, fetches.cancelled)
	}
}

func TestACancelledFetchIsReportedAsCancelled(t *testing.T) {
	fetches := &fetchRecord{}
	pageFetcher := pageFetcherFor(t, egressproxytest.Start(t), nil, fetches)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	wasFetched := wasFetchedBy(ctx, pageFetcher, requestFor(t, http.MethodGet,
		"https://site.example/page"))

	if wasFetched || len(fetches.failures) != 0 ||
		!slices.Equal(fetches.cancelled, []string{"https://site.example/page"}) {
		t.Fatalf("fetched %v, failures %v, cancelled %v",
			wasFetched, fetches.failures, fetches.cancelled)
	}
}

func wasFetchedBy(
	ctx context.Context,
	pageFetcher *tlsclient.PageFetcher,
	request chromenavigation.Request,
) bool {
	page, wasFetched := pageFetcher.Fetch(ctx, request)
	if wasFetched {
		_ = page.Body.Close()
	}
	return wasFetched
}

func pageFetcherFor(
	t *testing.T,
	egressProxy *egressproxytest.EgressProxy,
	trustedRoots *x509.CertPool,
	fetches *fetchRecord,
) *tlsclient.PageFetcher {
	t.Helper()
	pageFetcher, err := tlsclient.New(
		egressProxy.URL(),
		trustedRoots,
		time.Minute,
		tlsclient.Observers{fetches},
	)
	if err != nil {
		t.Fatal(err)
	}
	return pageFetcher
}

func trustedRootsOf(origin *httptest.Server) *x509.CertPool {
	trustedRoots := x509.NewCertPool()
	trustedRoots.AddCert(origin.Certificate())
	return trustedRoots
}

func requestFor(t *testing.T, method, rawAddress string) chromenavigation.Request {
	t.Helper()
	address, err := url.Parse(rawAddress)
	if err != nil {
		t.Fatal(err)
	}
	return chromenavigation.New("").RequestFor(method, address, http.Header{})
}

func fetchedPageOf(
	t *testing.T,
	pageFetcher *tlsclient.PageFetcher,
	method, rawAddress string,
) (int, http.Header, string) {
	t.Helper()
	page, wasFetched := pageFetcher.Fetch(t.Context(), requestFor(t, method, rawAddress))
	if !wasFetched {
		t.Fatalf("%s %s not fetched", method, rawAddress)
	}
	defer func() { _ = page.Body.Close() }()
	body, err := io.ReadAll(page.Body)
	if err != nil {
		t.Fatal(err)
	}
	return page.StatusCode, page.Header, string(body)
}

func rawOrigin(t *testing.T) (string, <-chan []string) {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	headerNames := make(chan []string, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = connection.Close() }()
		headerNames <- headerNamesFrom(bufio.NewReader(connection))
		_, _ = io.WriteString(connection, "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
	}()
	return listener.Addr().String(), headerNames
}

func headerNamesFrom(reader *bufio.Reader) []string {
	var names []string
	_, _ = reader.ReadString('\n')
	for {
		line, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(line) == "" {
			return names
		}
		name, _, _ := strings.Cut(line, ":")
		names = append(names, name)
	}
}
