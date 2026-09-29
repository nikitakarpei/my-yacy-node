package requestrelay_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	spamassessmenthttpheader "github.com/nikitakarpei/yacy-rwi-node/spamassessment/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
)

const pageBody = "<html><title>Cheap pills</title><p>Buy cheap pills now</p></html>"

const pageByteCeiling = 1000

var (
	requestArrivedAt = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	spamAssessment   = spamassessment.Assessment{
		Score:        0.9,
		Threshold:    0.8,
		ModelVersion: "2026-09",
	}
	errEgressDown = errors.New("egress proxy down")
	testLimits    = requestrelay.Limits{
		PageByteCeiling:       pageByteCeiling,
		MaxPagesReadAtOnce:    4,
		ResponseHeaderTimeout: time.Second,
		RelayIdleTimeout:      500 * time.Millisecond,
	}
)

type fakeClock struct {
	mutex       sync.Mutex
	now         time.Time
	expirations chan func()
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: requestArrivedAt, expirations: make(chan func(), 64)}
}

func (c *fakeClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.now
}

func (c *fakeClock) After(_ time.Duration, expire func()) func() {
	var wasStopped atomic.Bool
	c.expirations <- func() {
		if !wasStopped.Load() {
			expire()
		}
	}
	return func() { wasStopped.Store(true) }
}

func (c *fakeClock) passTo(elapsed time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.now = requestArrivedAt.Add(elapsed)
}

type upstreamResponse struct {
	status  int
	headers http.Header
	body    func(ctx context.Context) io.Reader
	before  func()
}

func htmlUpstreamResponseWith(body string) upstreamResponse {
	return upstreamResponse{
		status:  http.StatusOK,
		headers: http.Header{"Content-Type": {"text/html; charset=utf-8"}},
		body:    func(context.Context) io.Reader { return strings.NewReader(body) },
	}
}

type fakeEgress struct {
	mutex            sync.Mutex
	upstreamResponse upstreamResponse
	requests         []*http.Request
	failure          error
}

func (e *fakeEgress) RoundTrip(request *http.Request) (*http.Response, error) {
	e.mutex.Lock()
	e.requests = append(e.requests, request)
	e.mutex.Unlock()
	if e.failure != nil {
		return nil, e.failure
	}
	if e.upstreamResponse.before != nil {
		e.upstreamResponse.before()
	}
	if e.upstreamResponse.body == nil {
		<-request.Context().Done()
		return nil, fmt.Errorf("silent egress: %w", request.Context().Err())
	}
	return &http.Response{
		StatusCode: e.upstreamResponse.status,
		Header:     e.upstreamResponse.headers,
		Body:       io.NopCloser(e.upstreamResponse.body(request.Context())),
	}, nil
}

type stallingBody struct {
	ctx context.Context
}

func (b stallingBody) Read([]byte) (int, error) {
	<-b.ctx.Done()
	return 0, fmt.Errorf("stalled body: %w", b.ctx.Err())
}

type cancellableBody struct {
	ctx  context.Context
	body io.Reader
}

func (b cancellableBody) Read(chunk []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, fmt.Errorf("cancelled body: %w", err)
	}
	//nolint:wrapcheck // io.EOF reaches the caller of a reader unwrapped
	return b.body.Read(chunk)
}

type failingBody struct {
	failure error
}

func (b failingBody) Read([]byte) (int, error) {
	return 0, b.failure
}

type assessedBody struct {
	body    []byte
	headers http.Header
}

type fakeAssessmentRunner struct {
	mutex         sync.Mutex
	assessedPages []assessedBody
	outcome       assessmentgate.Outcome
	during        func()
}

func (a *fakeAssessmentRunner) Assess(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
	_ time.Time,
) (spamassessment.Assessment, assessmentgate.Outcome) {
	a.mutex.Lock()
	a.assessedPages = append(a.assessedPages, assessedBody{body: body, headers: responseHeaders})
	a.mutex.Unlock()
	if a.during != nil {
		a.during()
	}
	return spamAssessment, a.outcome
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
	mutex              sync.Mutex
	refusals           []requestrelay.RefusalReason
	skippedAssessments []requestrelay.SkipReason
	upstreamFailures   []upstreamFailure
	incompleteCauses   []requestrelay.IncompleteResponseCause
	closedRequests     int
	assessed           []spamassessment.Assessment
	readingSlotWaits   []time.Duration
}

func (r *observerRecord) PageAssessed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.assessed = append(r.assessed, assessment)
}

func (r *observerRecord) ReadingSlotWaited(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.readingSlotWaits = append(r.readingSlotWaits, slotWait)
}

func (r *observerRecord) AssessmentSkipped(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason requestrelay.SkipReason,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.skippedAssessments = append(r.skippedAssessments, reason)
}

func (r *observerRecord) RequestRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason requestrelay.RefusalReason,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.refusals = append(r.refusals, reason)
}

type upstreamFailure struct {
	failure requestrelay.UpstreamResponseFailure
	cause   error
}

func (r *observerRecord) UpstreamResponseFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	failure requestrelay.UpstreamResponseFailure,
	cause error,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.upstreamFailures = append(r.upstreamFailures, upstreamFailure{failure, cause})
}

func (r *observerRecord) ResponseLeftIncomplete(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	incompleteResponseCause requestrelay.IncompleteResponseCause,
	_ error,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.incompleteCauses = append(r.incompleteCauses, incompleteResponseCause)
}

func (r *observerRecord) ClientClosedRequest(context.Context, canonicalurl.CanonicalURL) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.closedRequests++
}

type responderFixture struct {
	egress           *fakeEgress
	assessmentRunner *fakeAssessmentRunner
	clock            *fakeClock
	observers        *observerRecord
	limits           requestrelay.Limits
}

func newResponderFixture() *responderFixture {
	return &responderFixture{
		egress:           &fakeEgress{upstreamResponse: htmlUpstreamResponseWith(pageBody)},
		assessmentRunner: &fakeAssessmentRunner{},
		clock:            newFakeClock(),
		observers:        &observerRecord{},
		limits:           testLimits,
	}
}

func (f *responderFixture) responder() *requestrelay.Responder {
	return requestrelay.New(
		f.egress,
		f.assessmentRunner,
		requestrelay.Observers{f.observers},
		f.limits,
		f.clock,
	)
}

func (f *responderFixture) respondTo(
	t *testing.T,
	method string,
	requestHeaders http.Header,
) *recordedResponse {
	t.Helper()
	response := &recordedResponse{}
	f.responder().RespondTo(t.Context(), method, pageAddress(t), requestHeaders, response)
	return response
}

func (f *responderFixture) respondInBackgroundTo(
	t *testing.T,
	responder *requestrelay.Responder,
) <-chan *recordedResponse {
	t.Helper()
	responses := make(chan *recordedResponse, 1)
	address := pageAddress(t)
	go func() {
		response := &recordedResponse{}
		responder.RespondTo(context.Background(), http.MethodGet, address, http.Header{}, response)
		responses <- response
	}()
	return responses
}

func pageAddress(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	address, err := canonicalurl.CanonicalURLOf("http://site.example/page")
	if err != nil {
		t.Fatal(err)
	}
	return address
}

func TestAnHTMLPageCarriesTheVerdictAndItsBody(t *testing.T) {
	fixture := newResponderFixture()

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if response.status != http.StatusOK ||
		response.headers.Get(
			"Spam-Assessment",
		) != spamassessmenthttpheader.ValueOf(
			spamAssessment,
		) ||
		response.String() != pageBody {
		t.Fatalf("response %d %v %q", response.status, response.headers, response.String())
	}
	if string(fixture.assessmentRunner.assessedPages[0].body) != pageBody {
		t.Fatalf("assessed body %q", fixture.assessmentRunner.assessedPages[0].body)
	}
}

func TestTheEgressIsAskedForTheAddressWithTheRelayedRequestHeaders(t *testing.T) {
	fixture := newResponderFixture()

	fixture.respondTo(
		t,
		http.MethodGet,
		http.Header{"User-Agent": {"crawler"}, "Cookie": {"session=1"}},
	)

	request := fixture.egress.requests[0]
	if request.URL.String() != "http://site.example/page" ||
		request.Header.Get("User-Agent") != "crawler" ||
		request.Header.Get("Cookie") != "" {
		t.Fatalf("upstream request %s %v", request.URL, request.Header)
	}
}

func TestTheVerdictOfTheOriginIsReplaced(t *testing.T) {
	fixture := newResponderFixture()
	fixture.egress.upstreamResponse.headers.Set("Spam-Assessment", "clean")

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if verdicts := response.headers.Values("Spam-Assessment"); len(verdicts) != 1 ||
		verdicts[0] != spamassessmenthttpheader.ValueOf(spamAssessment) {
		t.Fatalf("verdicts %v", verdicts)
	}
}

func TestAnUpstreamResponseWhoseAssessmentIsSkippedIsRelayedWithoutAVerdict(t *testing.T) {
	upstreamResponses := map[requestrelay.SkipReason]struct {
		method           string
		upstreamResponse upstreamResponse
	}{
		requestrelay.NotHTML: {http.MethodGet, upstreamResponse{
			status:  http.StatusOK,
			headers: http.Header{"Content-Type": {"image/png"}, "Spam-Assessment": {"clean"}},
			body:    func(context.Context) io.Reader { return strings.NewReader("png") },
		}},
		requestrelay.Not2xx: {
			http.MethodGet,
			upstreamResponse{
				status: http.StatusFound,
				headers: http.Header{
					"Content-Type": {"text/html"},
					"Location":     {"http://other.example/"},
				},
				body: func(context.Context) io.Reader { return strings.NewReader("png") },
			},
		},
		requestrelay.HeadRequest: {http.MethodHead, htmlUpstreamResponseWith("png")},
	}
	for reason, skippedResponse := range upstreamResponses {
		t.Run(string(reason), func(t *testing.T) {
			fixture := newResponderFixture()
			fixture.egress.upstreamResponse = skippedResponse.upstreamResponse

			response := fixture.respondTo(t, skippedResponse.method, http.Header{})

			if response.status != skippedResponse.upstreamResponse.status ||
				response.headers.Get("Spam-Assessment") != "" ||
				response.String() != "png" ||
				fixture.observers.skippedAssessments[0] != reason {
				t.Fatalf(
					"response %d %v %q, reasons %v",
					response.status,
					response.headers,
					response.String(),
					fixture.observers.skippedAssessments,
				)
			}
		})
	}
}

func TestAGzipPageIsAssessedDecodedAndRelayedAsTheOriginSentIt(t *testing.T) {
	fixture := newResponderFixture()
	var encodedBody bytes.Buffer
	writer := gzip.NewWriter(&encodedBody)
	_, _ = writer.Write([]byte(pageBody))
	_ = writer.Close()
	fixture.egress.upstreamResponse = htmlUpstreamResponseWith(encodedBody.String())
	fixture.egress.upstreamResponse.headers.Set("Content-Encoding", "gzip")

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if response.headers.Get("Spam-Assessment") == "" ||
		response.headers.Get("Content-Encoding") != "gzip" ||
		response.String() != encodedBody.String() {
		t.Fatalf("response %v %q", response.headers, response.String())
	}
	if string(fixture.assessmentRunner.assessedPages[0].body) != pageBody {
		t.Fatalf("assessed body %q", fixture.assessmentRunner.assessedPages[0].body)
	}
}

func TestAPageThatCannotBeDecodedGivesBadGateway(t *testing.T) {
	for encoding, reason := range map[string]requestrelay.RefusalReason{
		"br":   requestrelay.UndecodableEncoding,
		"gzip": requestrelay.UndecodableBody,
	} {
		t.Run(encoding, func(t *testing.T) {
			fixture := newResponderFixture()
			fixture.egress.upstreamResponse = htmlUpstreamResponseWith("zipped")
			fixture.egress.upstreamResponse.headers.Set("Content-Encoding", encoding)

			response := fixture.respondTo(t, http.MethodGet, http.Header{})

			if response.status != http.StatusBadGateway || fixture.observers.refusals[0] != reason {
				t.Fatalf("status %d, refusals %v", response.status, fixture.observers.refusals)
			}
		})
	}
}

func TestAPageOverTheByteCeilingIsAssessedOnItsFirstBytesAndRelayedWhole(t *testing.T) {
	fixture := newResponderFixture()
	body := "<p>" + strings.Repeat("x", pageByteCeiling) + "</p>"
	fixture.egress.upstreamResponse = htmlUpstreamResponseWith(body)
	fixture.egress.upstreamResponse.headers.Set("Content-Length", "1007")

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if response.headers.Get("Content-Length") != "1007" || response.String() != body {
		t.Fatalf("response %v, %d bytes", response.headers, response.Len())
	}
	if assessedBytes := len(
		fixture.assessmentRunner.assessedPages[0].body,
	); assessedBytes != pageByteCeiling {
		t.Fatalf("assessed %d bytes", assessedBytes)
	}
}

func TestAPageOverTheByteCeilingAssessedPastTheEndOfReadingIsRelayedWhole(t *testing.T) {
	fixture := newResponderFixture()
	body := "<p>" + strings.Repeat("x", pageByteCeiling) + "</p>"
	fixture.egress.upstreamResponse = htmlUpstreamResponseWith(body)
	fixture.egress.upstreamResponse.body = func(ctx context.Context) io.Reader {
		return cancellableBody{ctx: ctx, body: strings.NewReader(body)}
	}
	fixture.assessmentRunner.during = func() { (<-fixture.clock.expirations)() }

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if response.wasAborted || response.String() != body {
		t.Fatalf("aborted %v, %d bytes", response.wasAborted, response.Len())
	}
}

func TestAWholePageOfUnknownLengthGetsItsLength(t *testing.T) {
	fixture := newResponderFixture()

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if response.headers.Get("Content-Length") != "65" {
		t.Fatalf("content length %q", response.headers.Get("Content-Length"))
	}
}

func TestAFailedEgressGivesBadGateway(t *testing.T) {
	fixture := newResponderFixture()
	fixture.egress.failure = errEgressDown

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if failures := fixture.observers.upstreamFailures; response.status != http.StatusBadGateway ||
		failures[0].failure != requestrelay.NoResponse || !errors.Is(failures[0].cause, errEgressDown) {
		t.Fatalf("status %d, failures %v", response.status, failures)
	}
}

func TestASilentEgressGivesGatewayTimeoutWhenReadingEnds(t *testing.T) {
	fixture := newResponderFixture()
	fixture.egress.upstreamResponse.body = nil
	responses := fixture.respondInBackgroundTo(t, fixture.responder())

	(<-fixture.clock.expirations)()

	if response := <-responses; response.status != http.StatusGatewayTimeout {
		t.Fatalf("status %d", response.status)
	}
}

func TestAPageWhoseBodyStallsGivesGatewayTimeoutWhenReadingEnds(t *testing.T) {
	fixture := newResponderFixture()
	fixture.egress.upstreamResponse.body = func(ctx context.Context) io.Reader { return stallingBody{ctx} }
	responses := fixture.respondInBackgroundTo(t, fixture.responder())

	(<-fixture.clock.expirations)()

	if response := <-responses; response.status != http.StatusGatewayTimeout ||
		fixture.observers.refusals[0] != requestrelay.PageReadDeadline {
		t.Fatalf("status %d, refusals %v", response.status, fixture.observers.refusals)
	}
}

func TestAPageWhoseBodyPrefixFailsToReadGivesBadGateway(t *testing.T) {
	fixture := newResponderFixture()
	fixture.egress.upstreamResponse.body = func(context.Context) io.Reader {
		return io.MultiReader(strings.NewReader("<p>short</p>"), failingBody{io.ErrUnexpectedEOF})
	}

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if failures := fixture.observers.upstreamFailures; response.status != http.StatusBadGateway ||
		failures[0].failure != requestrelay.BodyPrefixReadFailed ||
		!errors.Is(failures[0].cause, io.ErrUnexpectedEOF) {
		t.Fatalf("status %d, failures %v", response.status, failures)
	}
}

func TestASkippedUpstreamResponseWhoseBodyFailsToReadIsLeftIncomplete(t *testing.T) {
	fixture := newResponderFixture()
	fixture.egress.upstreamResponse = upstreamResponse{
		status:  http.StatusOK,
		headers: http.Header{"Content-Type": {"image/png"}},
		body: func(context.Context) io.Reader {
			return io.MultiReader(strings.NewReader("png"), failingBody{io.ErrUnexpectedEOF})
		},
	}

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if response.status != http.StatusOK || response.String() != "png" || !response.wasAborted ||
		!slices.Equal(fixture.observers.incompleteCauses,
			[]requestrelay.IncompleteResponseCause{requestrelay.BodyRestReadFailed}) {
		t.Fatalf("response %d %q, aborted %v, causes %v", response.status, response.String(),
			response.wasAborted, fixture.observers.incompleteCauses)
	}
}

func TestARequestThatTheClientClosesBeforeTheHeadersGetsNoResponse(t *testing.T) {
	fixture := newResponderFixture()
	fixture.egress.upstreamResponse.body = nil
	clientCtx, closeRequest := context.WithCancel(t.Context())
	closeRequest()
	response := &recordedResponse{}

	fixture.responder().
		RespondTo(clientCtx, http.MethodGet, pageAddress(t), http.Header{}, response)

	if response.status != 0 || fixture.observers.closedRequests != 1 ||
		len(fixture.observers.upstreamFailures) != 0 {
		t.Fatalf("status %d, closed requests %d, upstream failures %v", response.status,
			fixture.observers.closedRequests, fixture.observers.upstreamFailures)
	}
}

func TestARequestThatTheClientClosesDuringTheBodyIsLeftIncomplete(t *testing.T) {
	fixture := newResponderFixture()
	clientCtx, closeRequest := context.WithCancel(t.Context())
	fixture.egress.upstreamResponse = upstreamResponse{
		status:  http.StatusOK,
		headers: http.Header{"Content-Type": {"image/png"}},
		body: func(context.Context) io.Reader {
			closeRequest()
			return io.MultiReader(strings.NewReader("png"), failingBody{io.ErrUnexpectedEOF})
		},
	}
	response := &recordedResponse{}

	fixture.responder().
		RespondTo(clientCtx, http.MethodGet, pageAddress(t), http.Header{}, response)

	if !response.wasAborted || fixture.observers.closedRequests != 0 ||
		!slices.Equal(fixture.observers.incompleteCauses,
			[]requestrelay.IncompleteResponseCause{requestrelay.ClientClosedRequest}) {
		t.Fatalf("aborted %v, closed requests %d, causes %v", response.wasAborted,
			fixture.observers.closedRequests, fixture.observers.incompleteCauses)
	}
}

func TestAResponseThatTheClientStopsTakingIsLeftIncomplete(t *testing.T) {
	fixture := newResponderFixture()
	response := &recordedResponse{writeFailure: errors.New("broken pipe")}

	fixture.responder().
		RespondTo(t.Context(), http.MethodGet, pageAddress(t), http.Header{}, response)

	if !response.wasAborted || !slices.Equal(fixture.observers.incompleteCauses,
		[]requestrelay.IncompleteResponseCause{requestrelay.ClientClosedRequest}) {
		t.Fatalf("aborted %v, causes %v", response.wasAborted, fixture.observers.incompleteCauses)
	}
}

func TestAPageWhoseRestStallsIsLeftIncompleteAfterTheIdleTimeout(t *testing.T) {
	fixture := newResponderFixture()
	fixture.limits.PageByteCeiling = 8
	fixture.egress.upstreamResponse.body = func(ctx context.Context) io.Reader {
		return io.MultiReader(strings.NewReader(pageBody), stallingBody{ctx})
	}
	responses := fixture.respondInBackgroundTo(t, fixture.responder())

	<-fixture.clock.expirations
	<-fixture.clock.expirations
	(<-fixture.clock.expirations)()

	if response := <-responses; response.status != http.StatusOK || response.String() != pageBody ||
		!response.wasAborted || !slices.Equal(fixture.observers.incompleteCauses,
		[]requestrelay.IncompleteResponseCause{requestrelay.RelayIdleTimeout}) {
		t.Fatalf("response %d %q, aborted %v, causes %v", response.status, response.String(),
			response.wasAborted, fixture.observers.incompleteCauses)
	}
}

func TestAPageWaitingForAReadingSlotIsRelayedOnceTheSlotFrees(t *testing.T) {
	fixture := newResponderFixture()
	fixture.limits.MaxPagesReadAtOnce = 1
	started, released := make(chan struct{}), make(chan struct{})
	fixture.assessmentRunner.during = heldOnFirstAssessment(started, released)
	responder := fixture.responder()
	heldResponses := fixture.respondInBackgroundTo(t, responder)
	<-started
	waitingResponses := fixture.respondInBackgroundTo(t, responder)
	<-fixture.clock.expirations
	<-fixture.clock.expirations

	close(released)

	if waitingResponse := <-waitingResponses; waitingResponse.status != http.StatusOK ||
		waitingResponse.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("waiting response %d %v", waitingResponse.status, waitingResponse.headers)
	}
	if heldResponse := <-heldResponses; heldResponse.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("held response %v", heldResponse.headers)
	}
}

func TestAPageWithoutAReadingSlotWhenTheHeadersAreDueGivesServiceUnavailable(t *testing.T) {
	fixture := newResponderFixture()
	fixture.limits.MaxPagesReadAtOnce = 1
	started, released := make(chan struct{}), make(chan struct{})
	fixture.assessmentRunner.during = heldOnFirstAssessment(started, released)
	responder := fixture.responder()
	heldResponses := fixture.respondInBackgroundTo(t, responder)
	<-started
	waitingResponses := fixture.respondInBackgroundTo(t, responder)
	<-fixture.clock.expirations

	(<-fixture.clock.expirations)()
	waitingResponse := <-waitingResponses
	close(released)

	if waitingResponse.status != http.StatusServiceUnavailable ||
		waitingResponse.headers.Get("Retry-After") != "1" {
		t.Fatalf("waiting response %d %v", waitingResponse.status, waitingResponse.headers)
	}
	if heldResponse := <-heldResponses; heldResponse.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("held response %v", heldResponse.headers)
	}
	if refusals := fixture.observers.refusals; len(refusals) != 1 ||
		refusals[0] != requestrelay.SlotWaitDeadline {
		t.Fatalf("refusals %v", refusals)
	}
	if slotWaits := fixture.observers.readingSlotWaits; len(slotWaits) != 2 {
		t.Fatalf("reading slot waits %v, want one for each page", slotWaits)
	}
}

func heldOnFirstAssessment(started chan<- struct{}, released <-chan struct{}) func() {
	var assessments atomic.Int32
	return func() {
		if assessments.Add(1) == 1 {
			close(started)
			<-released
		}
	}
}

func TestAnUpstreamResponseThatStartsAfterTheHeadersAreDueGivesGatewayTimeout(t *testing.T) {
	fixture := newResponderFixture()
	fixture.egress.upstreamResponse.before = func() { fixture.clock.passTo(1100 * time.Millisecond) }

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if response.status != http.StatusGatewayTimeout ||
		fixture.observers.refusals[0] != requestrelay.PageReadDeadline {
		t.Fatalf("status %d, refusals %v", response.status, fixture.observers.refusals)
	}
}

func TestAPageThatWasNotAssessedIsRefused(t *testing.T) {
	refusals := map[assessmentgate.Outcome]struct {
		reason requestrelay.RefusalReason
		status int
	}{
		assessmentgate.SlotWaitDeadline: {
			requestrelay.SlotWaitDeadline,
			http.StatusServiceUnavailable,
		},
		assessmentgate.AssessmentDeadline: {
			requestrelay.AssessmentDeadline,
			http.StatusGatewayTimeout,
		},
		assessmentgate.Panicked: {
			requestrelay.UnassessedPage,
			http.StatusServiceUnavailable,
		},
	}
	for outcome, refusal := range refusals {
		fixture := newResponderFixture()
		fixture.assessmentRunner.outcome = outcome

		response := fixture.respondTo(t, http.MethodGet, http.Header{})

		if response.status != refusal.status || response.headers.Get("Spam-Assessment") != "" ||
			fixture.observers.refusals[0] != refusal.reason {
			t.Errorf("outcome %v: response %d %v, refusals %v",
				outcome, response.status, response.headers, fixture.observers.refusals)
		}
	}
}

func TestAPageAssessedPastTheResponseHeaderDeadlineGivesGatewayTimeout(t *testing.T) {
	fixture := newResponderFixture()
	fixture.assessmentRunner.during = func() { fixture.clock.passTo(time.Second) }

	response := fixture.respondTo(t, http.MethodGet, http.Header{})

	if response.status != http.StatusGatewayTimeout ||
		response.headers.Get("Spam-Assessment") != "" ||
		fixture.observers.refusals[0] != requestrelay.AssessmentDeadline {
		t.Fatalf(
			"response %d %v, refusals %v",
			response.status,
			response.headers,
			fixture.observers.refusals,
		)
	}
}

func TestAWaitPreferenceOfZeroGivesGatewayTimeoutWithoutAskingTheEgress(t *testing.T) {
	fixture := newResponderFixture()

	response := fixture.respondTo(t, http.MethodGet, http.Header{"Prefer": {"wait=0"}})

	if response.status != http.StatusGatewayTimeout || len(fixture.egress.requests) != 0 {
		t.Fatalf("status %d, egress requests %d", response.status, len(fixture.egress.requests))
	}
}
