package requestrelay_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxiedrequest"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtripcancel"
)

const responseHeaderTimeout = 10 * time.Second

var (
	requestArrivedAt = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	errEgressDown    = errors.New("egress proxy down")
)

type fakeClock struct{}

func (fakeClock) Now() time.Time { return requestArrivedAt }

type failingBody struct {
	failure error
}

func (b failingBody) Read([]byte) (int, error) {
	return 0, b.failure
}

type closeRecordingBody struct {
	io.Reader
	wasClosed bool
}

func (b *closeRecordingBody) Close() error {
	b.wasClosed = true
	return nil
}

type fakeUpstream struct {
	requests []proxiedrequest.Request
	headers  http.Header
	body     *closeRecordingBody
	failure  error
	before   func()
}

func (u *fakeUpstream) RoundTrip(
	_ context.Context,
	request proxiedrequest.Request,
) (*http.Response, error) {
	u.requests = append(u.requests, request)
	if u.before != nil {
		u.before()
	}
	if u.failure != nil {
		return nil, u.failure
	}
	return &http.Response{StatusCode: http.StatusOK, Header: u.headers, Body: u.body}, nil
}

func upstreamWithBody(body io.Reader) *fakeUpstream {
	return &fakeUpstream{
		headers: http.Header{"Content-Type": {"image/png"}},
		body:    &closeRecordingBody{Reader: body},
	}
}

type recordedResponse struct {
	status  int
	headers http.Header
	bytes.Buffer
	writeFailure error
	wasAborted   bool
}

func (r *recordedResponse) SendHeaders(status int, headers http.Header) {
	r.status, r.headers = status, headers
}

func (r *recordedResponse) Write(chunk []byte) (int, error) {
	if r.writeFailure != nil {
		return 0, r.writeFailure
	}
	return r.Buffer.Write(chunk) //nolint:wrapcheck // the buffer never fails
}

func (r *recordedResponse) Abort() { r.wasAborted = true }

type observerRecord struct {
	incompleteCauses []requestrelay.IncompleteResponseCause
	closedRequests   int
}

func (r *observerRecord) ResponseLeftIncomplete(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	incompleteResponseCause requestrelay.IncompleteResponseCause,
	_ error,
) {
	r.incompleteCauses = append(r.incompleteCauses, incompleteResponseCause)
}

func (r *observerRecord) ClientClosedRequest(context.Context, canonicalurl.CanonicalURL) {
	r.closedRequests++
}

func respondTo(
	t *testing.T,
	ctx context.Context,
	upstream *fakeUpstream,
	requestHeaders http.Header,
	response *recordedResponse,
) *observerRecord {
	t.Helper()
	address, err := canonicalurl.CanonicalURLOf("http://site.example/page")
	if err != nil {
		t.Fatal(err)
	}
	observers := &observerRecord{}
	requestrelay.New(upstream, responseHeaderTimeout, fakeClock{}, requestrelay.Observers{observers}).
		RespondTo(ctx, http.MethodGet, address, requestHeaders, response)
	return observers
}

func TestTheUpstreamResponseIsSentWithoutHopByHopHeadersAndClosed(t *testing.T) {
	upstream := upstreamWithBody(strings.NewReader("png"))
	upstream.headers.Set("Connection", "close")
	response := &recordedResponse{}

	respondTo(t, t.Context(), upstream, http.Header{}, response)

	if response.status != http.StatusOK || response.headers.Get("Content-Type") != "image/png" ||
		response.headers.Get("Connection") != "" || response.String() != "png" ||
		!upstream.body.wasClosed {
		t.Fatalf("response %d %v %q, closed %v", response.status, response.headers,
			response.String(), upstream.body.wasClosed)
	}
}

func TestTheUpstreamIsAskedWithTheRelayedHeadersAndWhenTheHeadersAreDue(t *testing.T) {
	upstream := upstreamWithBody(strings.NewReader("png"))

	respondTo(t, t.Context(), upstream,
		http.Header{"User-Agent": {"crawler"}, "Cookie": {"session=1"}, "Prefer": {"wait=4"}},
		&recordedResponse{})

	request := upstream.requests[0]
	if request.Method != http.MethodGet || request.Address.String() != "http://site.example/page" ||
		request.Headers.Get("User-Agent") != "crawler" || request.Headers.Get("Cookie") != "" ||
		!request.HeadersDeadline.Equal(requestArrivedAt.Add(4*time.Second)) {
		t.Fatalf("proxied request %+v", request)
	}
}

func TestAFailedUpstreamGivesBadGateway(t *testing.T) {
	upstream := &fakeUpstream{failure: errEgressDown}
	response := &recordedResponse{}

	respondTo(t, t.Context(), upstream, http.Header{}, response)

	if response.status != http.StatusBadGateway {
		t.Fatalf("status %d", response.status)
	}
}

func TestARequestThatTheClientClosesBeforeTheHeadersGetsNoResponse(t *testing.T) {
	clientCtx, closeRequest := context.WithCancel(t.Context())
	upstream := &fakeUpstream{failure: errEgressDown, before: closeRequest}
	response := &recordedResponse{}

	observers := respondTo(t, clientCtx, upstream, http.Header{}, response)

	if response.status != 0 || observers.closedRequests != 1 {
		t.Fatalf("status %d, closed requests %d", response.status, observers.closedRequests)
	}
}

func TestAnUpstreamResponseLeftIncompleteIsAbortedWithItsCause(t *testing.T) {
	for cause, failure := range map[requestrelay.IncompleteResponseCause]error{
		requestrelay.BodyRestReadFailed: io.ErrUnexpectedEOF,
		requestrelay.RelayIdleTimeout:   roundtripcancel.ErrIdleTimeoutPassed,
	} {
		t.Run(string(cause), func(t *testing.T) {
			upstream := upstreamWithBody(
				io.MultiReader(strings.NewReader("png"), failingBody{failure}))
			response := &recordedResponse{}

			observers := respondTo(t, t.Context(), upstream, http.Header{}, response)

			if response.String() != "png" || !response.wasAborted ||
				!slices.Equal(
					observers.incompleteCauses,
					[]requestrelay.IncompleteResponseCause{cause},
				) {
				t.Fatalf("response %q, aborted %v, causes %v",
					response.String(), response.wasAborted, observers.incompleteCauses)
			}
		})
	}
}

func TestAResponseThatTheClientStopsTakingIsLeftIncomplete(t *testing.T) {
	for name, respond := range map[string]func(*testing.T, *fakeUpstream) *observerRecord{
		"write failed": func(t *testing.T, upstream *fakeUpstream) *observerRecord {
			t.Helper()
			return respondTo(t, t.Context(), upstream, http.Header{},
				&recordedResponse{writeFailure: errors.New("broken pipe")})
		},
		"request closed": func(t *testing.T, upstream *fakeUpstream) *observerRecord {
			t.Helper()
			clientCtx, closeRequest := context.WithCancel(t.Context())
			upstream.before = closeRequest
			return respondTo(t, clientCtx, upstream, http.Header{}, &recordedResponse{})
		},
	} {
		t.Run(name, func(t *testing.T) {
			upstream := upstreamWithBody(
				io.MultiReader(strings.NewReader("png"), failingBody{io.ErrUnexpectedEOF}))

			observers := respond(t, upstream)

			if !slices.Equal(observers.incompleteCauses,
				[]requestrelay.IncompleteResponseCause{requestrelay.ClientClosedRequest}) {
				t.Fatalf("causes %v", observers.incompleteCauses)
			}
		})
	}
}
