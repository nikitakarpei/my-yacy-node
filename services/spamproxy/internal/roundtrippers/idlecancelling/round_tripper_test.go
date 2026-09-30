package idlecancelling_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtripcancel"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/idlecancelling"
)

const idleTimeout = 30 * time.Second

var errEgressDown = errors.New("egress proxy down")

type armedTimer struct {
	timeout time.Duration
	expire  func()
	stopped bool
}

type fakeClock struct {
	armedTimers []*armedTimer
}

func (c *fakeClock) After(timeout time.Duration, expire func()) func() {
	armed := &armedTimer{timeout: timeout, expire: expire}
	c.armedTimers = append(c.armedTimers, armed)
	return func() { armed.stopped = true }
}

type fakeUpstream struct {
	request *http.Request
	body    io.Reader
	failure error
}

func (u *fakeUpstream) RoundTrip(request *http.Request) (*http.Response, error) {
	u.request = request
	if u.failure != nil {
		return nil, u.failure
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(u.body)}, nil
}

type stallingBody struct {
	clock    *fakeClock
	upstream *fakeUpstream
}

func (b stallingBody) Read([]byte) (int, error) {
	b.clock.armedTimers[len(b.clock.armedTimers)-1].expire()
	<-b.upstream.request.Context().Done()
	return 0, fmt.Errorf("stalled body: %w", b.upstream.request.Context().Err())
}

func requestFor(t *testing.T) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"http://site.example/",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func TestEachReadWaitsAtMostTheIdleTimeoutAndOnlyWhileItReads(t *testing.T) {
	clock, upstream := &fakeClock{}, &fakeUpstream{body: strings.NewReader("page")}
	roundTripper := idlecancelling.New(upstream, idleTimeout, clock)

	response, err := roundTripper.RoundTrip(requestFor(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)

	if err != nil || string(body) != "page" || upstream.request.Context().Err() != nil {
		t.Fatalf("body %q, error %v, round trip %v", body, err, upstream.request.Context().Err())
	}
	for position, armed := range clock.armedTimers {
		if !armed.stopped || armed.timeout != idleTimeout {
			t.Fatalf("timer %d: timeout %v, stopped %v", position, armed.timeout, armed.stopped)
		}
	}
	if len(clock.armedTimers) < 2 {
		t.Fatalf("armed %d timers, want one per read", len(clock.armedTimers))
	}
}

func TestAReadThatStallsPastTheIdleTimeoutFailsForTheIdleTimeout(t *testing.T) {
	clock, upstream := &fakeClock{}, &fakeUpstream{}
	upstream.body = stallingBody{clock: clock, upstream: upstream}
	roundTripper := idlecancelling.New(upstream, idleTimeout, clock)

	response, err := roundTripper.RoundTrip(requestFor(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	_, err = io.ReadAll(response.Body)

	if !errors.Is(err, roundtripcancel.ErrIdleTimeoutPassed) {
		t.Fatalf("error %v", err)
	}
}

func TestClosingTheResponseEndsTheRoundTrip(t *testing.T) {
	upstream := &fakeUpstream{body: strings.NewReader("page")}
	roundTripper := idlecancelling.New(upstream, idleTimeout, &fakeClock{})
	response, err := roundTripper.RoundTrip(requestFor(t))
	if err != nil {
		t.Fatal(err)
	}

	_ = response.Body.Close()

	if upstream.request.Context().Err() == nil {
		t.Fatal("round trip still open after close")
	}
}

func TestAFailedUpstreamIsPassedOnUnchanged(t *testing.T) {
	roundTripper := idlecancelling.New(
		&fakeUpstream{failure: errEgressDown},
		idleTimeout,
		&fakeClock{},
	)

	response, err := roundTripper.RoundTrip(requestFor(t))
	if response != nil {
		_ = response.Body.Close()
	}

	if !errors.Is(err, errEgressDown) {
		t.Fatalf("error %v", err)
	}
}
