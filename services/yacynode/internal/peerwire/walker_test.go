package peerwire_test

import (
	"io"
	"maps"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	walkBudget      = 30 * time.Second
	formMemoryBytes = 1 << 20
)

var transferRequest = peerwire.Request{
	Path: "/yacy/transferRWI.html",
	Form: url.Values{"iam": {"peer"}},
}

type clock struct {
	guard       sync.Mutex
	at          time.Time
	timers      map[int]pendingTimer
	nextTimerID int
	armed       chan struct{}
}

type pendingTimer struct {
	firesAt time.Time
	expire  func()
}

func startClock() *clock {
	return &clock{
		at:     time.Unix(1_700_000_000, 0).UTC(),
		timers: make(map[int]pendingTimer),
		armed:  make(chan struct{}, 16),
	}
}

func (c *clock) Now() time.Time {
	c.guard.Lock()
	defer c.guard.Unlock()

	return c.at
}

func (c *clock) After(timeout time.Duration, expire func()) (stop func()) {
	c.guard.Lock()
	defer c.guard.Unlock()

	timerID := c.nextTimerID
	c.nextTimerID++
	c.timers[timerID] = pendingTimer{firesAt: c.at.Add(timeout), expire: expire}
	c.armed <- struct{}{}

	return func() {
		c.guard.Lock()
		defer c.guard.Unlock()

		delete(c.timers, timerID)
	}
}

func (c *clock) advance(elapsed time.Duration) {
	c.guard.Lock()
	c.at = c.at.Add(elapsed)
	var dueTimers []func()
	for timerID, timer := range c.timers {
		if !timer.firesAt.After(c.at) {
			dueTimers = append(dueTimers, timer.expire)
			delete(c.timers, timerID)
		}
	}
	c.guard.Unlock()

	for _, expire := range dueTimers {
		expire()
	}
}

func baseURLOf(t *testing.T, server *httptest.Server, scheme peerwire.Scheme) peerwire.BaseURL {
	t.Helper()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	host, err := yacymodel.ParseHost(parsed.Hostname())
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	return peerwire.BaseURL{Scheme: scheme, Host: host, Port: yacymodel.Port(port)}
}

func replyingServer(reply string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(reply))
	}))
}

type unusedPeerTransports struct{}

func (unusedPeerTransports) TransportFor(yacymodel.PeerIdentity) http.RoundTripper {
	panic("a walk without an identity asked for a peer transport")
}

func hasResult(fields yacyproto.ResponseFields) bool { return fields["result"] != "" }

func TestTryAtReturnsTheAcceptedReplyFromTheGivenBaseURL(t *testing.T) {
	server := replyingServer("result=ok\n")
	defer server.Close()
	baseURL := baseURLOf(t, server, peerwire.HTTP)

	answer, answered := peerwire.NewWalker(server.Client(), unusedPeerTransports{}, startClock()).
		TryAt(
			t.Context(),
			transferRequest,
			baseURL,
			hasResult,
			walkBudget,
		)

	if !answered || answer.BaseURL != baseURL || answer.Fields["result"] != "ok" {
		t.Fatalf("answer = %v, %v, want ok from %v", answer, answered, baseURL)
	}
}

type recordingPeerTransports struct {
	transport  http.RoundTripper
	identities []yacymodel.PeerIdentity
}

func (r *recordingPeerTransports) TransportFor(identity yacymodel.PeerIdentity) http.RoundTripper {
	r.identities = append(r.identities, identity)

	return r.transport
}

func TestTryAtGoesToAnHTTPSBaseURLThroughTheTransportForTheIdentityOfThePeer(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()
	peerTransports := &recordingPeerTransports{transport: server.Client().Transport}
	request := transferRequest
	request.Callee = peerwire.Callee{
		Hash:     peerkeytest.Key(0).Hash(),
		Identity: yacymodel.Some(peerkeytest.Key(0).Identity()),
	}

	_, answered := peerwire.NewWalker(http.DefaultClient, peerTransports, startClock()).TryAt(
		t.Context(),
		request,
		baseURLOf(t, server, peerwire.HTTPS),
		hasResult,
		walkBudget,
	)

	want := []yacymodel.PeerIdentity{peerkeytest.Key(0).Identity()}
	if !answered || !slices.Equal(peerTransports.identities, want) {
		t.Fatalf("answered = %v through transports for %v, want ok through the transport for %v",
			answered, peerTransports.identities, want)
	}
}

func TestTryAtGoesToAnHTTPSBaseURLOfAPeerWithoutIdentityThroughTheGivenClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()

	_, answered := peerwire.NewWalker(server.Client(), unusedPeerTransports{}, startClock()).TryAt(
		t.Context(),
		transferRequest,
		baseURLOf(t, server, peerwire.HTTPS),
		hasResult,
		walkBudget,
	)

	if !answered {
		t.Fatal("an https base URL without identity did not answer through the given client")
	}
}

func TestTryAtHasNoAnswerWhenTheReplyIsNotAccepted(t *testing.T) {
	server := replyingServer("<html lang=\"en\">\n")
	defer server.Close()

	_, answered := peerwire.NewWalker(server.Client(), unusedPeerTransports{}, startClock()).TryAt(
		t.Context(),
		transferRequest,
		baseURLOf(t, server, peerwire.HTTP),
		hasResult,
		walkBudget,
	)

	if answered {
		t.Fatal("a reply the caller does not accept answered")
	}
}

func TestTryInTurnPostsTheFormAndReturnsTheAcceptedReply(t *testing.T) {
	var (
		gotPath        string
		gotContentType string
		gotForm        url.Values
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType, _, _ = mime.ParseMediaType(r.Header.Get("Content-Type"))
		//nolint:gosec // G120: the test server reads one small form.
		if err := r.ParseMultipartForm(formMemoryBytes); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotForm = r.PostForm
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()
	baseURL := baseURLOf(t, server, peerwire.HTTP)

	answer, answered := peerwire.NewWalker(server.Client(), unusedPeerTransports{}, startClock()).
		TryInTurn(
			t.Context(),
			transferRequest,
			[]peerwire.BaseURL{baseURL},
			hasResult,
			walkBudget,
		)

	if !answered {
		t.Fatal("expected an answer")
	}
	if gotPath != "/yacy/transferRWI.html" {
		t.Errorf("path = %q, want /yacy/transferRWI.html", gotPath)
	}
	if gotContentType != "multipart/form-data" {
		t.Errorf("content type = %q", gotContentType)
	}
	if gotForm.Get("iam") != "peer" {
		t.Errorf("form iam = %q, want peer", gotForm.Get("iam"))
	}
	if answer.Fields["result"] != "ok" {
		t.Errorf("result = %q, want ok", answer.Fields["result"])
	}
	if answer.BaseURL != baseURL {
		t.Errorf("base URL = %v, want %v", answer.BaseURL, baseURL)
	}
}

func TestTryInTurnPostsEveryFormValueUnchanged(t *testing.T) {
	sentForm := url.Values{
		"indexes": {"AAAAAAAAAAAA{h=abc,a=1}\nBBBBBBBBBBBB{h=def,a=2}"},
		"title":   {"Заголовок & 100% = a+b"},
		"tag":     {"first", "second"},
	}
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:gosec // G120: the test server reads one small form.
		if err := r.ParseMultipartForm(formMemoryBytes); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotForm = r.PostForm
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()

	_, answered := peerwire.NewWalker(server.Client(), unusedPeerTransports{}, startClock()).
		TryInTurn(
			t.Context(),
			peerwire.Request{Path: "/yacy/transferURL.html", Form: sentForm},
			[]peerwire.BaseURL{baseURLOf(t, server, peerwire.HTTP)},
			hasResult,
			walkBudget,
		)

	if !answered {
		t.Fatal("expected an answer")
	}
	if !maps.EqualFunc(gotForm, sentForm, slices.Equal) {
		t.Errorf("form = %v, want %v", gotForm, sentForm)
	}
}

func TestTryInTurnReachesAnHTTPSBaseURL(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()
	baseURL := baseURLOf(t, server, peerwire.HTTPS)

	answer, answered := peerwire.NewWalker(server.Client(), unusedPeerTransports{}, startClock()).
		TryInTurn(
			t.Context(),
			transferRequest,
			[]peerwire.BaseURL{baseURL},
			hasResult,
			walkBudget,
		)

	if !answered || answer.BaseURL.Scheme != peerwire.HTTPS {
		t.Fatalf("answer = %v, %v, want an answer over https", answer, answered)
	}
}

func TestTryInTurnMovesOnPastBaseURLsThatGiveNoAcceptedReply(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream connect error", http.StatusBadGateway)
	}))
	defer failing.Close()
	answering := replyingServer("result=ok\n")
	defer answering.Close()
	redirecting := httptest.NewServer(
		http.RedirectHandler(answering.URL, http.StatusPermanentRedirect),
	)
	defer redirecting.Close()
	foreign := replyingServer("<html lang=\"en\">\n")
	defer foreign.Close()
	closed := replyingServer("result=ok\n")
	closedBaseURL := baseURLOf(t, closed, peerwire.HTTP)
	closed.Close()

	answeringBaseURL := baseURLOf(t, answering, peerwire.HTTP)
	baseURLs := []peerwire.BaseURL{
		closedBaseURL,
		baseURLOf(t, failing, peerwire.HTTP),
		baseURLOf(t, redirecting, peerwire.HTTP),
		baseURLOf(t, foreign, peerwire.HTTP),
		answeringBaseURL,
	}

	answer, answered := peerwire.NewWalker(http.DefaultClient, unusedPeerTransports{}, startClock()).
		TryInTurn(
			t.Context(),
			transferRequest,
			baseURLs,
			hasResult,
			walkBudget,
		)

	if !answered || answer.BaseURL != answeringBaseURL {
		t.Fatalf("answer = %v, %v, want the last base URL", answer, answered)
	}
}

func TestTryInTurnHasNoAnswerWhenEveryBaseURLFails(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failing.Close()

	_, answered := peerwire.NewWalker(failing.Client(), unusedPeerTransports{}, startClock()).
		TryInTurn(
			t.Context(),
			transferRequest,
			[]peerwire.BaseURL{baseURLOf(t, failing, peerwire.HTTP)},
			hasResult,
			walkBudget,
		)

	if answered {
		t.Fatal("expected no answer")
	}
}

func hangingServer(arrived chan<- struct{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		arrived <- struct{}{}
		<-r.Context().Done()
	}))
}

func TestTryInTurnGivesEachAttemptAShareOfTheTimeLeft(t *testing.T) {
	arrived := make(chan struct{}, 1)
	hanging := hangingServer(arrived)
	defer hanging.Close()
	answering := replyingServer("result=ok\n")
	defer answering.Close()
	answeringBaseURL := baseURLOf(t, answering, peerwire.HTTP)
	timeline := startClock()

	answers := make(chan bool, 1)
	go func() {
		_, answered := peerwire.NewWalker(http.DefaultClient, unusedPeerTransports{}, timeline).
			TryInTurn(
				t.Context(),
				transferRequest,
				[]peerwire.BaseURL{
					baseURLOf(t, hanging, peerwire.HTTP),
					answeringBaseURL,
				},
				hasResult,
				walkBudget,
			)
		answers <- answered
	}()
	<-timeline.armed
	<-arrived
	timeline.advance(walkBudget / 2)

	if !<-answers {
		t.Fatal("expected the second base URL to answer once the first used its share")
	}
}

func TestTryInTurnStopsWhenTheBudgetIsSpent(t *testing.T) {
	arrived := make(chan struct{}, 1)
	hanging := hangingServer(arrived)
	defer hanging.Close()
	var laterBaseURLAsked bool
	later := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		laterBaseURLAsked = true
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer later.Close()
	timeline := startClock()

	answers := make(chan bool, 1)
	go func() {
		_, answered := peerwire.NewWalker(http.DefaultClient, unusedPeerTransports{}, timeline).
			TryInTurn(
				t.Context(),
				transferRequest,
				[]peerwire.BaseURL{
					baseURLOf(t, hanging, peerwire.HTTP),
					baseURLOf(t, later, peerwire.HTTP),
				},
				hasResult,
				walkBudget,
			)
		answers <- answered
	}()
	<-timeline.armed
	<-arrived
	timeline.advance(walkBudget)

	if <-answers {
		t.Fatal("expected no answer after the budget was spent")
	}
	if laterBaseURLAsked {
		t.Error("a base URL was asked after the budget was spent")
	}
}

func TestTryInTurnMeasuresTheAnswerTimeOfTheAnsweringAttempt(t *testing.T) {
	timeline := startClock()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		timeline.advance(2 * time.Second)
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()

	answer, _ := peerwire.NewWalker(server.Client(), unusedPeerTransports{}, timeline).TryInTurn(
		t.Context(),
		transferRequest,
		[]peerwire.BaseURL{baseURLOf(t, server, peerwire.HTTP)},
		hasResult,
		walkBudget,
	)

	if answer.Time != 2*time.Second {
		t.Errorf("answer time = %v, want 2s", answer.Time)
	}
}
