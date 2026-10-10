package proxyintake_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/chromenavigation"
	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/proxyintake"
)

const pageBody = "<html>page</html>"

var errOriginStopped = errors.New("origin stopped")

type scriptedPageFetcher struct {
	lock      sync.Mutex
	requests  []chromenavigation.Request
	fails     bool
	readFails bool
}

func (f *scriptedPageFetcher) Fetch(
	_ context.Context,
	request chromenavigation.Request,
) (*http.Response, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.requests = append(f.requests, request)
	if f.fails {
		return nil, false
	}
	var body io.Reader = strings.NewReader(pageBody)
	if f.readFails {
		body = io.MultiReader(body, failingReader{})
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Header: http.Header{
			"Connection": {"X-Hop"},
			"X-Hop":      {"1"},
			"Keep-Alive": {"timeout=5"},
			"X-Page":     {"kept"},
		},
		Body: io.NopCloser(body),
	}, true
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errOriginStopped }

type intakeRecord struct {
	lock                     sync.Mutex
	methods                  []string
	targets                  []string
	incompleteResponseCauses []proxyintake.IncompleteResponseCause
}

func (r *intakeRecord) MethodRefused(_ context.Context, method string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.methods = append(r.methods, method)
}

func (r *intakeRecord) TargetRefused(_ context.Context, target string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.targets = append(r.targets, target)
}

func (r *intakeRecord) ResponseLeftIncomplete(
	_ context.Context,
	_ string,
	incompleteResponseCause proxyintake.IncompleteResponseCause,
	_ error,
) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.incompleteResponseCauses = append(r.incompleteResponseCauses, incompleteResponseCause)
}

func TestAnAbsoluteWebAddressIsFetchedAsChromeAndRelayed(t *testing.T) {
	pageFetcher := &scriptedPageFetcher{}
	server := intakeServer(t, pageFetcher, &intakeRecord{})

	answer := proxiedAnswerTo(t, server, "GET https://site.example/page?q=1 HTTP/1.1")

	if answer.status != http.StatusNotFound || answer.body != pageBody || answer.readErr != nil ||
		answer.headers.Get("X-Page") != "kept" || answer.headers.Get("X-Hop") != "" ||
		answer.headers.Get("Keep-Alive") != "" {
		t.Fatalf("answer %+v", answer)
	}
	if _, found := answer.headers["Content-Type"]; found {
		t.Fatalf("headers %v", answer.headers)
	}
	request := pageFetcher.requests[0]
	if request.Address.String() != "https://site.example/page?q=1" ||
		!slices.Contains(request.HeaderFields, chromenavigation.HeaderField{
			Name: "referer", Value: "https://www.google.com/",
		}) {
		t.Fatalf("fetched request %+v", request)
	}
}

func TestOnlyGetAndHeadAreFetched(t *testing.T) {
	for _, requestLine := range []string{
		"POST http://site.example/page HTTP/1.1",
		"CONNECT site.example:443 HTTP/1.1",
	} {
		pageFetcher := &scriptedPageFetcher{}
		refusals := &intakeRecord{}
		server := intakeServer(t, pageFetcher, refusals)

		answer := proxiedAnswerTo(t, server, requestLine)

		if answer.status != http.StatusMethodNotAllowed || len(pageFetcher.requests) != 0 ||
			len(refusals.methods) != 1 {
			t.Fatalf("%s: status %d, requests %v, refused methods %v", requestLine,
				answer.status, pageFetcher.requests, refusals.methods)
		}
	}
}

func TestATargetThatIsNotAnAbsoluteWebAddressIsABadRequest(t *testing.T) {
	for _, target := range []string{"/page", "ftp://site.example/page"} {
		pageFetcher := &scriptedPageFetcher{}
		refusals := &intakeRecord{}
		server := intakeServer(t, pageFetcher, refusals)

		answer := proxiedAnswerTo(t, server, "GET "+target+" HTTP/1.1")

		if answer.status != http.StatusBadRequest || len(pageFetcher.requests) != 0 ||
			!slices.Equal(refusals.targets, []string{target}) {
			t.Fatalf("%s: status %d, requests %v, refused targets %v",
				target, answer.status, pageFetcher.requests, refusals.targets)
		}
	}
}

func TestAPageThatIsNotFetchedIsABadGateway(t *testing.T) {
	server := intakeServer(t, &scriptedPageFetcher{fails: true}, &intakeRecord{})

	answer := proxiedAnswerTo(t, server, "GET http://site.example/page HTTP/1.1")

	if answer.status != http.StatusBadGateway {
		t.Fatalf("status %d", answer.status)
	}
}

func TestAPageThatStopsMidwayEndsTheResponseEarly(t *testing.T) {
	observations := &intakeRecord{}
	server := intakeServer(t, &scriptedPageFetcher{readFails: true}, observations)

	answer := proxiedAnswerTo(t, server, "GET http://site.example/page HTTP/1.1")

	if answer.body != pageBody || answer.readErr == nil {
		t.Fatalf("answer %+v", answer)
	}
	if !slices.Equal(observations.incompleteResponseCauses,
		[]proxyintake.IncompleteResponseCause{proxyintake.PageReadFailed}) {
		t.Fatalf("incomplete response causes %v", observations.incompleteResponseCauses)
	}
}

func intakeServer(
	t *testing.T,
	pageFetcher *scriptedPageFetcher,
	observations *intakeRecord,
) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(proxyintake.New(
		chromenavigation.New("https://www.google.com/"),
		pageFetcher,
		proxyintake.Observers{observations},
	))
	t.Cleanup(server.Close)
	return server
}

type proxiedAnswer struct {
	status  int
	headers http.Header
	body    string
	readErr error
}

func proxiedAnswerTo(t *testing.T, server *httptest.Server, requestLine string) proxiedAnswer {
	t.Helper()
	var dialer net.Dialer
	connection, err := dialer.DialContext(
		t.Context(),
		"tcp",
		strings.TrimPrefix(server.URL, "http://"),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	if _, err := io.WriteString(
		connection,
		requestLine+"\r\nHost: site.example\r\n\r\n",
	); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, readErr := io.ReadAll(response.Body)
	return proxiedAnswer{
		status:  response.StatusCode,
		headers: response.Header,
		body:    string(body),
		readErr: readErr,
	}
}
