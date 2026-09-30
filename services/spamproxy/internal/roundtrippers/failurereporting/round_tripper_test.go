package failurereporting_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/failurereporting"
)

var errEgressProxy = errors.New("egress proxy failed")

type scriptedEgress struct {
	connects      bool
	writesRequest bool
	responds      bool
}

func (e scriptedEgress) RoundTrip(request *http.Request) (*http.Response, error) {
	trace := httptrace.ContextClientTrace(request.Context())
	if e.connects {
		trace.GotConn(httptrace.GotConnInfo{})
	}
	if e.writesRequest {
		trace.WroteRequest(httptrace.WroteRequestInfo{})
	}
	if !e.responds {
		return nil, errEgressProxy
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

type failureRecord struct {
	steps  []failurereporting.Step
	causes []error
}

func (r *failureRecord) RoundTripFailed(
	_ context.Context,
	_ string,
	step failurereporting.Step,
	cause error,
) {
	r.steps = append(r.steps, step)
	r.causes = append(r.causes, cause)
}

func TestAFailedUpstreamRequestIsReportedWithTheStepThatFailed(t *testing.T) {
	for step, egress := range map[failurereporting.Step]scriptedEgress{
		failurereporting.ConnectFailed:      {},
		failurereporting.RequestWriteFailed: {connects: true},
		failurereporting.HeadersReadFailed:  {connects: true, writesRequest: true},
	} {
		t.Run(string(step), func(t *testing.T) {
			failures := &failureRecord{}
			roundTripper := failurereporting.New(egress, failurereporting.Observers{failures})

			err := failureOf(roundTripper, requestWith(t, t.Context()))

			if !errors.Is(err, errEgressProxy) ||
				!slices.Equal(failures.steps, []failurereporting.Step{step}) ||
				!errors.Is(failures.causes[0], errEgressProxy) {
				t.Fatalf("error %v, steps %v", err, failures.steps)
			}
		})
	}
}

func TestAnAnsweredUpstreamRequestIsNotReported(t *testing.T) {
	failures := &failureRecord{}
	roundTripper := failurereporting.New(
		scriptedEgress{connects: true, writesRequest: true, responds: true},
		failurereporting.Observers{failures},
	)

	response, err := roundTripper.RoundTrip(requestWith(t, t.Context()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK || len(failures.steps) != 0 {
		t.Fatalf("status %d, steps %v", response.StatusCode, failures.steps)
	}
}

func TestACancelledUpstreamRequestIsNotReported(t *testing.T) {
	failures := &failureRecord{}
	roundTripper := failurereporting.New(scriptedEgress{}, failurereporting.Observers{failures})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := failureOf(roundTripper, requestWith(t, ctx))

	if err == nil || len(failures.steps) != 0 {
		t.Fatalf("error %v, steps %v", err, failures.steps)
	}
}

func failureOf(roundTripper *failurereporting.RoundTripper, request *http.Request) error {
	response, err := roundTripper.RoundTrip(request)
	if response != nil {
		_ = response.Body.Close()
	}
	return err
}

func requestWith(t *testing.T, ctx context.Context) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://site.example/", nil)
	if err != nil {
		t.Fatal(err)
	}
	return request
}
