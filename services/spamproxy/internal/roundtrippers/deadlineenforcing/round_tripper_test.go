package deadlineenforcing_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxiedrequest"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/deadlineenforcing"
)

var (
	requestArrivedAt = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	errEgressDown    = errors.New("egress proxy down")
)

type armedTimer struct {
	timeout time.Duration
	expire  func()
	stopped bool
}

type fakeClock struct {
	armedTimers chan *armedTimer
}

func newFakeClock() *fakeClock {
	return &fakeClock{armedTimers: make(chan *armedTimer, 4)}
}

func (c *fakeClock) Now() time.Time { return requestArrivedAt }

func (c *fakeClock) After(timeout time.Duration, expire func()) func() {
	armed := &armedTimer{timeout: timeout, expire: expire}
	c.armedTimers <- armed
	return func() { armed.stopped = true }
}

type fakeUpstream struct {
	requests []proxiedrequest.Request
	contexts []context.Context
	isSilent bool
	failure  error
}

func (u *fakeUpstream) RoundTrip(
	ctx context.Context,
	request proxiedrequest.Request,
) (*http.Response, error) {
	u.requests = append(u.requests, request)
	u.contexts = append(u.contexts, ctx)
	if u.isSilent {
		<-ctx.Done()
		return nil, fmt.Errorf("silent upstream: %w", ctx.Err())
	}
	if u.failure != nil {
		return nil, u.failure
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"text/html"}},
		Body:       io.NopCloser(strings.NewReader("page")),
	}, nil
}

type refusalRecord struct {
	reasons []deadlineenforcing.RefusalReason
}

func (r *refusalRecord) RequestRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason deadlineenforcing.RefusalReason,
) {
	r.reasons = append(r.reasons, reason)
}

func requestDueAfter(t *testing.T, wait time.Duration) proxiedrequest.Request {
	t.Helper()
	address, err := canonicalurl.CanonicalURLOf("http://site.example/page")
	if err != nil {
		t.Fatal(err)
	}
	return proxiedrequest.Request{
		Method:          http.MethodGet,
		Address:         address,
		HeadersDeadline: requestArrivedAt.Add(wait),
	}
}

func statusOf(response *http.Response, err error) int {
	if err != nil {
		return 0
	}
	_ = response.Body.Close()
	return response.StatusCode
}

func failureOf(response *http.Response, err error) error {
	if response != nil {
		_ = response.Body.Close()
	}
	return err
}

func TestARequestWithoutTimeForTheHeadersGivesGatewayTimeoutWithoutAskingTheUpstream(t *testing.T) {
	upstream, refusals := &fakeUpstream{}, &refusalRecord{}
	roundTripper := deadlineenforcing.New(upstream, newFakeClock(),
		deadlineenforcing.Observers{refusals})

	status := statusOf(roundTripper.RoundTrip(t.Context(), requestDueAfter(t, 0)))

	if status != http.StatusGatewayTimeout || len(upstream.requests) != 0 ||
		len(refusals.reasons) != 1 || refusals.reasons[0] != deadlineenforcing.WaitTooShort {
		t.Fatalf("status %d, upstream requests %d, refusals %v",
			status, len(upstream.requests), refusals.reasons)
	}
}

func TestAResponseBeforeTheDeadlineIsReturnedAndReadable(t *testing.T) {
	upstream, clock := &fakeUpstream{}, newFakeClock()
	roundTripper := deadlineenforcing.New(upstream, clock, deadlineenforcing.Observers{})

	response, err := roundTripper.RoundTrip(t.Context(), requestDueAfter(t, 3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	deadline := <-clock.armedTimers
	deadline.expire()
	body, err := io.ReadAll(response.Body)

	if err != nil || string(body) != "page" || deadline.timeout != 3*time.Second ||
		!deadline.stopped {
		t.Fatalf("body %q, error %v, timeout %v, stopped %v",
			body, err, deadline.timeout, deadline.stopped)
	}
}

func TestClosingTheResponseEndsTheRoundTrip(t *testing.T) {
	upstream := &fakeUpstream{}
	roundTripper := deadlineenforcing.New(upstream, newFakeClock(), deadlineenforcing.Observers{})
	response, err := roundTripper.RoundTrip(t.Context(), requestDueAfter(t, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	roundTripCtx := upstream.contexts[0]
	wasEndedBeforeClose := roundTripCtx.Err() != nil

	_ = response.Body.Close()

	if wasEndedBeforeClose || roundTripCtx.Err() == nil {
		t.Fatalf("ended before close %v, after close %v", wasEndedBeforeClose, roundTripCtx.Err())
	}
}

func TestASilentUpstreamGivesGatewayTimeoutWhenTheHeadersAreDue(t *testing.T) {
	upstream, clock, refusals := &fakeUpstream{isSilent: true}, newFakeClock(), &refusalRecord{}
	roundTripper := deadlineenforcing.New(upstream, clock, deadlineenforcing.Observers{refusals})
	statuses := make(chan int, 1)
	request := requestDueAfter(t, time.Second)
	go func() { statuses <- statusOf(roundTripper.RoundTrip(context.Background(), request)) }()

	(<-clock.armedTimers).expire()

	if status := <-statuses; status != http.StatusGatewayTimeout ||
		len(refusals.reasons) != 1 || refusals.reasons[0] != deadlineenforcing.HeadersDeadline {
		t.Fatalf("status %d, refusals %v", status, refusals.reasons)
	}
}

func TestAFailedUpstreamIsPassedOnUnchanged(t *testing.T) {
	upstream, refusals := &fakeUpstream{failure: errEgressDown}, &refusalRecord{}
	roundTripper := deadlineenforcing.New(upstream, newFakeClock(),
		deadlineenforcing.Observers{refusals})

	err := failureOf(roundTripper.RoundTrip(t.Context(), requestDueAfter(t, time.Second)))

	if !errors.Is(err, errEgressDown) || len(refusals.reasons) != 0 {
		t.Fatalf("error %v, refusals %v", err, refusals.reasons)
	}
}

func TestARequestThatTheClientClosesIsPassedOnUnrefused(t *testing.T) {
	upstream, refusals := &fakeUpstream{isSilent: true}, &refusalRecord{}
	roundTripper := deadlineenforcing.New(upstream, newFakeClock(),
		deadlineenforcing.Observers{refusals})
	clientCtx, closeRequest := context.WithCancel(t.Context())
	closeRequest()

	err := failureOf(roundTripper.RoundTrip(clientCtx, requestDueAfter(t, time.Second)))

	if !errors.Is(err, context.Canceled) || len(refusals.reasons) != 0 {
		t.Fatalf("error %v, refusals %v", err, refusals.reasons)
	}
}
