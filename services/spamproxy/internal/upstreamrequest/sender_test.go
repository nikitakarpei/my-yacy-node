package upstreamrequest_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/upstreamrequest"
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
	steps  []upstreamrequest.Step
	causes []error
}

func (r *failureRecord) Failed(
	_ context.Context,
	_ string,
	step upstreamrequest.Step,
	cause error,
) {
	r.steps = append(r.steps, step)
	r.causes = append(r.causes, cause)
}

func TestAFailedUpstreamRequestIsReportedWithTheStepThatFailed(t *testing.T) {
	for step, egress := range map[upstreamrequest.Step]scriptedEgress{
		upstreamrequest.ConnectFailed:      {},
		upstreamrequest.RequestWriteFailed: {connects: true},
		upstreamrequest.HeadersReadFailed:  {connects: true, writesRequest: true},
	} {
		t.Run(string(step), func(t *testing.T) {
			failures := &failureRecord{}
			sender := upstreamrequest.New(egress, upstreamrequest.Observers{failures})

			err := failureOf(sender, requestWith(t, t.Context()))

			if !errors.Is(err, errEgressProxy) ||
				!slices.Equal(failures.steps, []upstreamrequest.Step{step}) ||
				!errors.Is(failures.causes[0], errEgressProxy) {
				t.Fatalf("error %v, steps %v", err, failures.steps)
			}
		})
	}
}

func TestAnAnsweredUpstreamRequestIsNotReported(t *testing.T) {
	failures := &failureRecord{}
	sender := upstreamrequest.New(
		scriptedEgress{connects: true, writesRequest: true, responds: true},
		upstreamrequest.Observers{failures},
	)

	response, err := sender.RoundTrip(requestWith(t, t.Context()))
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
	sender := upstreamrequest.New(scriptedEgress{}, upstreamrequest.Observers{failures})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := failureOf(sender, requestWith(t, ctx))

	if err == nil || len(failures.steps) != 0 {
		t.Fatalf("error %v, steps %v", err, failures.steps)
	}
}

func failureOf(sender *upstreamrequest.Sender, request *http.Request) error {
	response, err := sender.RoundTrip(request)
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
