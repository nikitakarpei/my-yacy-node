package peerwire_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const exchangeBudget = 30 * time.Second

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

func originOf(t *testing.T, server *httptest.Server, scheme peerwire.Scheme) peerwire.Origin {
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

	return peerwire.Origin{Scheme: scheme, Host: host, Port: yacymodel.Port(port)}
}

func replyingServer(reply string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(reply))
	}))
}

func hasResult(fields yacyproto.ResponseFields) bool { return fields["result"] != "" }

func TestExchangePostsTheFormAndReturnsTheAcceptedReply(t *testing.T) {
	var (
		gotPath        string
		gotContentType string
		gotForm        url.Values
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotForm = r.PostForm
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()
	origin := originOf(t, server, peerwire.HTTP)

	answer, answered := peerwire.NewFormExchange(server.Client(), startClock()).Exchange(
		t.Context(),
		transferRequest,
		[]peerwire.Origin{origin},
		hasResult,
		exchangeBudget,
	)

	if !answered {
		t.Fatal("expected an answer")
	}
	if gotPath != "/yacy/transferRWI.html" {
		t.Errorf("path = %q, want /yacy/transferRWI.html", gotPath)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("content type = %q", gotContentType)
	}
	if gotForm.Get("iam") != "peer" {
		t.Errorf("form iam = %q, want peer", gotForm.Get("iam"))
	}
	if answer.Fields["result"] != "ok" {
		t.Errorf("result = %q, want ok", answer.Fields["result"])
	}
	if answer.Origin != origin {
		t.Errorf("origin = %v, want %v", answer.Origin, origin)
	}
}

func TestExchangeReachesAnHTTPSOrigin(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()
	origin := originOf(t, server, peerwire.HTTPS)

	answer, answered := peerwire.NewFormExchange(server.Client(), startClock()).Exchange(
		t.Context(),
		transferRequest,
		[]peerwire.Origin{origin},
		hasResult,
		exchangeBudget,
	)

	if !answered || answer.Origin.Scheme != peerwire.HTTPS {
		t.Fatalf("answer = %v, %v, want an answer over https", answer, answered)
	}
}

func TestExchangeMovesOnPastOriginsThatGiveNoAcceptedReply(t *testing.T) {
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
	closedOrigin := originOf(t, closed, peerwire.HTTP)
	closed.Close()

	answeringOrigin := originOf(t, answering, peerwire.HTTP)
	origins := []peerwire.Origin{
		closedOrigin,
		originOf(t, failing, peerwire.HTTP),
		originOf(t, redirecting, peerwire.HTTP),
		originOf(t, foreign, peerwire.HTTP),
		answeringOrigin,
	}

	answer, answered := peerwire.NewFormExchange(http.DefaultClient, startClock()).Exchange(
		t.Context(),
		transferRequest,
		origins,
		hasResult,
		exchangeBudget,
	)

	if !answered || answer.Origin != answeringOrigin {
		t.Fatalf("answer = %v, %v, want the last origin", answer, answered)
	}
}

func TestExchangeHasNoAnswerWhenEveryOriginFails(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failing.Close()

	_, answered := peerwire.NewFormExchange(failing.Client(), startClock()).Exchange(
		t.Context(),
		transferRequest,
		[]peerwire.Origin{originOf(t, failing, peerwire.HTTP)},
		hasResult,
		exchangeBudget,
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

func TestExchangeGivesEachAttemptAShareOfTheTimeLeft(t *testing.T) {
	arrived := make(chan struct{}, 1)
	hanging := hangingServer(arrived)
	defer hanging.Close()
	answering := replyingServer("result=ok\n")
	defer answering.Close()
	answeringOrigin := originOf(t, answering, peerwire.HTTP)
	timeline := startClock()

	answers := make(chan bool, 1)
	go func() {
		_, answered := peerwire.NewFormExchange(http.DefaultClient, timeline).Exchange(
			t.Context(),
			transferRequest,
			[]peerwire.Origin{
				originOf(t, hanging, peerwire.HTTP),
				answeringOrigin,
			},
			hasResult,
			exchangeBudget,
		)
		answers <- answered
	}()
	<-timeline.armed
	<-arrived
	timeline.advance(exchangeBudget / 2)

	if !<-answers {
		t.Fatal("expected the second origin to answer once the first used its share")
	}
}

func TestExchangeStopsWhenTheBudgetIsSpent(t *testing.T) {
	arrived := make(chan struct{}, 1)
	hanging := hangingServer(arrived)
	defer hanging.Close()
	var laterOriginAsked bool
	later := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		laterOriginAsked = true
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer later.Close()
	timeline := startClock()

	answers := make(chan bool, 1)
	go func() {
		_, answered := peerwire.NewFormExchange(http.DefaultClient, timeline).Exchange(
			t.Context(),
			transferRequest,
			[]peerwire.Origin{
				originOf(t, hanging, peerwire.HTTP),
				originOf(t, later, peerwire.HTTP),
			},
			hasResult,
			exchangeBudget,
		)
		answers <- answered
	}()
	<-timeline.armed
	<-arrived
	timeline.advance(exchangeBudget)

	if <-answers {
		t.Fatal("expected no answer after the budget was spent")
	}
	if laterOriginAsked {
		t.Error("an origin was asked after the budget was spent")
	}
}

func TestExchangeMeasuresTheAnswerTimeOfTheAnsweringAttempt(t *testing.T) {
	timeline := startClock()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		timeline.advance(2 * time.Second)
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()

	answer, _ := peerwire.NewFormExchange(server.Client(), timeline).Exchange(
		t.Context(),
		transferRequest,
		[]peerwire.Origin{originOf(t, server, peerwire.HTTP)},
		hasResult,
		exchangeBudget,
	)

	if answer.Time != 2*time.Second {
		t.Errorf("answer time = %v, want 2s", answer.Time)
	}
}
