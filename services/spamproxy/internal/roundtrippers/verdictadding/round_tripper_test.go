package verdictadding_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	spamassessmenthttpheader "github.com/nikitakarpei/yacy-rwi-node/spamassessment/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxiedrequest"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtripcancel"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/verdictadding"
)

const pageBody = "<html><title>Cheap pills</title><p>Buy cheap pills now</p></html>"

const pageByteCeiling = 1000

var (
	requestArrivedAt = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	headersDeadline  = requestArrivedAt.Add(time.Second)
	spamAssessment   = spamassessment.Assessment{
		Score:        0.9,
		Threshold:    0.8,
		ModelVersion: "2026-09",
	}
	errEgressDown = errors.New("egress proxy down")
)

type fakeClock struct {
	mutex sync.Mutex
	now   time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.now
}

func (c *fakeClock) passTo(elapsed time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.now = requestArrivedAt.Add(elapsed)
}

type upstreamResponse struct {
	status  int
	headers http.Header
	body    func() io.Reader
}

func htmlUpstreamResponseWith(body string) upstreamResponse {
	return upstreamResponse{
		status:  http.StatusOK,
		headers: http.Header{"Content-Type": {"text/html; charset=utf-8"}},
		body:    func() io.Reader { return strings.NewReader(body) },
	}
}

type fakeUpstream struct {
	mutex            sync.Mutex
	upstreamResponse upstreamResponse
	requests         []*http.Request
	failure          error
}

func (u *fakeUpstream) RoundTrip(request *http.Request) (*http.Response, error) {
	u.mutex.Lock()
	u.requests = append(u.requests, request)
	u.mutex.Unlock()
	if u.failure != nil {
		return nil, u.failure
	}
	return &http.Response{
		StatusCode: u.upstreamResponse.status,
		Header:     u.upstreamResponse.headers.Clone(),
		Body:       io.NopCloser(u.upstreamResponse.body()),
	}, nil
}

type cancellingBody struct {
	cancelReading context.CancelCauseFunc
	cause         error
}

func (b cancellingBody) Read([]byte) (int, error) {
	b.cancelReading(b.cause)
	return 0, context.Canceled
}

type failingBody struct {
	failure error
}

func (b failingBody) Read([]byte) (int, error) {
	return 0, b.failure
}

type fakeAssessmentRunner struct {
	mutex         sync.Mutex
	assessedPages [][]byte
	outcome       assessmentgate.Outcome
	during        func()
}

func (a *fakeAssessmentRunner) Assess(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	body []byte,
	_ http.Header,
) (spamassessment.Assessment, assessmentgate.Outcome) {
	a.mutex.Lock()
	a.assessedPages = append(a.assessedPages, body)
	a.mutex.Unlock()
	if a.during != nil {
		a.during()
	}
	return spamAssessment, a.outcome
}

type observerRecord struct {
	mutex                  sync.Mutex
	refusals               []verdictadding.RefusalReason
	skippedAssessments     []verdictadding.SkipReason
	bodyPrefixReadFailures []error
	assessed               []spamassessment.Assessment
	readingSlotWaits       []time.Duration
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
	reason verdictadding.SkipReason,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.skippedAssessments = append(r.skippedAssessments, reason)
}

func (r *observerRecord) PageRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason verdictadding.RefusalReason,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.refusals = append(r.refusals, reason)
}

func (r *observerRecord) BodyPrefixReadFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.bodyPrefixReadFailures = append(r.bodyPrefixReadFailures, cause)
}

type verdictFixture struct {
	upstream           *fakeUpstream
	assessmentRunner   *fakeAssessmentRunner
	clock              *fakeClock
	observers          *observerRecord
	maxPagesReadAtOnce int
}

func newVerdictFixture() *verdictFixture {
	return &verdictFixture{
		upstream:           &fakeUpstream{upstreamResponse: htmlUpstreamResponseWith(pageBody)},
		assessmentRunner:   &fakeAssessmentRunner{},
		clock:              &fakeClock{now: requestArrivedAt},
		observers:          &observerRecord{},
		maxPagesReadAtOnce: 4,
	}
}

func (f *verdictFixture) roundTripper() *verdictadding.RoundTripper {
	return verdictadding.New(
		f.upstream,
		f.assessmentRunner,
		verdictadding.Limits{
			PageByteCeiling:    pageByteCeiling,
			MaxPagesReadAtOnce: f.maxPagesReadAtOnce,
		},
		f.clock,
		verdictadding.Observers{f.observers},
	)
}

type readResponse struct {
	status  int
	headers http.Header
	body    string
}

func (f *verdictFixture) roundTrip(t *testing.T, ctx context.Context, method string) readResponse {
	t.Helper()
	response, err := f.roundTripper().RoundTrip(ctx, requestWith(t, method))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	return readResponseOf(t, response)
}

func (f *verdictFixture) roundTripInBackground(
	t *testing.T,
	roundTripper *verdictadding.RoundTripper,
	ctx context.Context,
) <-chan readResponse {
	t.Helper()
	responses := make(chan readResponse, 1)
	request := requestWith(t, http.MethodGet)
	go func() {
		response, err := roundTripper.RoundTrip(ctx, request)
		if err != nil {
			responses <- readResponse{}
			return
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		responses <- readResponse{status: response.StatusCode, headers: response.Header, body: string(body)}
	}()
	return responses
}

func requestWith(t *testing.T, method string) proxiedrequest.Request {
	t.Helper()
	address, err := canonicalurl.CanonicalURLOf("http://site.example/page")
	if err != nil {
		t.Fatal(err)
	}
	return proxiedrequest.Request{
		Method:          method,
		Address:         address,
		Headers:         http.Header{"User-Agent": {"crawler"}},
		HeadersDeadline: headersDeadline,
	}
}

func failureOf(response *http.Response, err error) error {
	if response != nil {
		_ = response.Body.Close()
	}
	return err
}

func readResponseOf(t *testing.T, response *http.Response) readResponse {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return readResponse{status: response.StatusCode, headers: response.Header, body: string(body)}
}

func TestAnHTMLPageCarriesTheVerdictAndItsBody(t *testing.T) {
	fixture := newVerdictFixture()

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	wantVerdict := spamassessmenthttpheader.ValueOf(spamAssessment)
	if response.status != http.StatusOK || response.headers.Get("Spam-Assessment") != wantVerdict ||
		response.body != pageBody {
		t.Fatalf("response %d %v %q", response.status, response.headers, response.body)
	}
	if string(fixture.assessmentRunner.assessedPages[0]) != pageBody {
		t.Fatalf("assessed body %q", fixture.assessmentRunner.assessedPages[0])
	}
}

func TestTheUpstreamIsAskedForTheAddressWithTheRequestHeaders(t *testing.T) {
	fixture := newVerdictFixture()

	fixture.roundTrip(t, t.Context(), http.MethodGet)

	request := fixture.upstream.requests[0]
	if request.Method != http.MethodGet || request.URL.String() != "http://site.example/page" ||
		request.Header.Get("User-Agent") != "crawler" {
		t.Fatalf("upstream request %s %s %v", request.Method, request.URL, request.Header)
	}
}

func TestTheVerdictOfTheOriginIsReplaced(t *testing.T) {
	fixture := newVerdictFixture()
	fixture.upstream.upstreamResponse.headers.Set("Spam-Assessment", "clean")

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	if verdicts := response.headers.Values("Spam-Assessment"); len(verdicts) != 1 ||
		verdicts[0] != spamassessmenthttpheader.ValueOf(spamAssessment) {
		t.Fatalf("verdicts %v", verdicts)
	}
}

func TestAnUpstreamResponseWhoseAssessmentIsSkippedIsReturnedWithoutAVerdict(t *testing.T) {
	upstreamResponses := map[verdictadding.SkipReason]struct {
		method           string
		upstreamResponse upstreamResponse
	}{
		verdictadding.NotHTML: {http.MethodGet, upstreamResponse{
			status:  http.StatusOK,
			headers: http.Header{"Content-Type": {"image/png"}, "Spam-Assessment": {"clean"}},
			body:    func() io.Reader { return strings.NewReader("png") },
		}},
		verdictadding.Not2xx: {
			http.MethodGet,
			upstreamResponse{
				status: http.StatusFound,
				headers: http.Header{
					"Content-Type": {"text/html"},
					"Location":     {"http://other.example/"},
				},
				body: func() io.Reader { return strings.NewReader("png") },
			},
		},
		verdictadding.HeadRequest: {http.MethodHead, htmlUpstreamResponseWith("png")},
	}
	for reason, skippedResponse := range upstreamResponses {
		t.Run(string(reason), func(t *testing.T) {
			fixture := newVerdictFixture()
			fixture.upstream.upstreamResponse = skippedResponse.upstreamResponse

			response := fixture.roundTrip(t, t.Context(), skippedResponse.method)

			if response.status != skippedResponse.upstreamResponse.status ||
				response.headers.Get("Spam-Assessment") != "" || response.body != "png" ||
				fixture.observers.skippedAssessments[0] != reason {
				t.Fatalf("response %d %v %q, reasons %v", response.status, response.headers,
					response.body, fixture.observers.skippedAssessments)
			}
		})
	}
}

func TestAGzipPageIsAssessedDecodedAndReturnedAsTheOriginSentIt(t *testing.T) {
	fixture := newVerdictFixture()
	var encodedBody bytes.Buffer
	writer := gzip.NewWriter(&encodedBody)
	_, _ = writer.Write([]byte(pageBody))
	_ = writer.Close()
	fixture.upstream.upstreamResponse = htmlUpstreamResponseWith(encodedBody.String())
	fixture.upstream.upstreamResponse.headers.Set("Content-Encoding", "gzip")

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	if response.headers.Get("Spam-Assessment") == "" ||
		response.headers.Get("Content-Encoding") != "gzip" ||
		response.body != encodedBody.String() {
		t.Fatalf("response %v %q", response.headers, response.body)
	}
	if string(fixture.assessmentRunner.assessedPages[0]) != pageBody {
		t.Fatalf("assessed body %q", fixture.assessmentRunner.assessedPages[0])
	}
}

func TestAPageThatCannotBeDecodedGivesBadGateway(t *testing.T) {
	for encoding, reason := range map[string]verdictadding.RefusalReason{
		"br":   verdictadding.UndecodableEncoding,
		"gzip": verdictadding.UndecodableBody,
	} {
		t.Run(encoding, func(t *testing.T) {
			fixture := newVerdictFixture()
			fixture.upstream.upstreamResponse = htmlUpstreamResponseWith("zipped")
			fixture.upstream.upstreamResponse.headers.Set("Content-Encoding", encoding)

			response := fixture.roundTrip(t, t.Context(), http.MethodGet)

			if response.status != http.StatusBadGateway || fixture.observers.refusals[0] != reason {
				t.Fatalf("status %d, refusals %v", response.status, fixture.observers.refusals)
			}
		})
	}
}

func TestAPageOverTheByteCeilingIsAssessedOnItsFirstBytesAndReturnedWhole(t *testing.T) {
	fixture := newVerdictFixture()
	body := "<p>" + strings.Repeat("x", pageByteCeiling) + "</p>"
	fixture.upstream.upstreamResponse = htmlUpstreamResponseWith(body)
	fixture.upstream.upstreamResponse.headers.Set("Content-Length", "1007")

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	if response.headers.Get("Content-Length") != "1007" || response.body != body {
		t.Fatalf("response %v, %d bytes", response.headers, len(response.body))
	}
	if assessedBytes := len(
		fixture.assessmentRunner.assessedPages[0],
	); assessedBytes != pageByteCeiling {
		t.Fatalf("assessed %d bytes", assessedBytes)
	}
}

func TestAWholePageOfUnknownLengthGetsItsLength(t *testing.T) {
	fixture := newVerdictFixture()

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	if response.headers.Get("Content-Length") != "65" {
		t.Fatalf("content length %q", response.headers.Get("Content-Length"))
	}
}

func TestAFailedUpstreamIsPassedOnUnchanged(t *testing.T) {
	fixture := newVerdictFixture()
	fixture.upstream.failure = errEgressDown

	err := failureOf(fixture.roundTripper().RoundTrip(t.Context(), requestWith(t, http.MethodGet)))

	if !errors.Is(err, errEgressDown) {
		t.Fatalf("error %v", err)
	}
}

func TestAPageWhoseBodyPrefixIsCutAtTheDeadlineGivesGatewayTimeout(t *testing.T) {
	fixture := newVerdictFixture()
	ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
	fixture.upstream.upstreamResponse.body = func() io.Reader {
		return cancellingBody{
			cancelReading: cancelRoundTrip,
			cause:         roundtripcancel.ErrHeadersDeadlinePassed,
		}
	}

	response := fixture.roundTrip(t, ctx, http.MethodGet)

	if response.status != http.StatusGatewayTimeout ||
		fixture.observers.refusals[0] != verdictadding.BodyPrefixReadDeadline {
		t.Fatalf("status %d, refusals %v", response.status, fixture.observers.refusals)
	}
}

func TestAPageWhoseBodyPrefixFailsToReadGivesBadGateway(t *testing.T) {
	fixture := newVerdictFixture()
	fixture.upstream.upstreamResponse.body = func() io.Reader {
		return io.MultiReader(strings.NewReader("<p>short</p>"), failingBody{io.ErrUnexpectedEOF})
	}

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	if failures := fixture.observers.bodyPrefixReadFailures; response.status != http.StatusBadGateway ||
		len(failures) != 1 ||
		!errors.Is(failures[0], io.ErrUnexpectedEOF) {
		t.Fatalf("status %d, failures %v", response.status, failures)
	}
}

func TestAPageWhoseClientLeavesDuringTheBodyPrefixGetsNoResponse(t *testing.T) {
	fixture := newVerdictFixture()
	ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
	fixture.upstream.upstreamResponse.body = func() io.Reader {
		return cancellingBody{cancelReading: cancelRoundTrip, cause: context.Canceled}
	}

	err := failureOf(fixture.roundTripper().RoundTrip(ctx, requestWith(t, http.MethodGet)))

	if !errors.Is(err, context.Canceled) || len(fixture.observers.refusals) != 0 {
		t.Fatalf("error %v, refusals %v", err, fixture.observers.refusals)
	}
}

func TestAPageWaitingForAReadingSlotIsAssessedOnceTheSlotFrees(t *testing.T) {
	fixture := newVerdictFixture()
	fixture.maxPagesReadAtOnce = 1
	started, released := make(chan struct{}), make(chan struct{})
	fixture.assessmentRunner.during = heldOnFirstAssessment(started, released)
	roundTripper := fixture.roundTripper()
	heldResponses := fixture.roundTripInBackground(t, roundTripper, t.Context())
	<-started
	waitingResponses := fixture.roundTripInBackground(t, roundTripper, t.Context())

	close(released)

	if waitingResponse := <-waitingResponses; waitingResponse.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("waiting response %d %v", waitingResponse.status, waitingResponse.headers)
	}
	if heldResponse := <-heldResponses; heldResponse.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("held response %v", heldResponse.headers)
	}
	if slotWaits := fixture.observers.readingSlotWaits; len(slotWaits) != 2 {
		t.Fatalf("reading slot waits %v, want one for each page", slotWaits)
	}
}

func TestAPageWithoutAReadingSlotAtTheDeadlineGivesServiceUnavailable(t *testing.T) {
	fixture := newVerdictFixture()
	fixture.maxPagesReadAtOnce = 1
	started, released := make(chan struct{}), make(chan struct{})
	fixture.assessmentRunner.during = heldOnFirstAssessment(started, released)
	roundTripper := fixture.roundTripper()
	heldResponses := fixture.roundTripInBackground(t, roundTripper, t.Context())
	<-started
	ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
	cancelRoundTrip(roundtripcancel.ErrHeadersDeadlinePassed)

	waitingResponse, err := roundTripper.RoundTrip(ctx, requestWith(t, http.MethodGet))
	close(released)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = waitingResponse.Body.Close() }()

	if waitingResponse.StatusCode != http.StatusServiceUnavailable ||
		waitingResponse.Header.Get("Retry-After") != "1" {
		t.Fatalf("waiting response %d %v", waitingResponse.StatusCode, waitingResponse.Header)
	}
	if heldResponse := <-heldResponses; heldResponse.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("held response %v", heldResponse.headers)
	}
	if refusals := fixture.observers.refusals; len(refusals) != 1 ||
		refusals[0] != verdictadding.SlotWaitDeadline {
		t.Fatalf("refusals %v", refusals)
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

func TestAPageThatWasNotAssessedIsRefused(t *testing.T) {
	refusals := map[assessmentgate.Outcome]struct {
		reason verdictadding.RefusalReason
		status int
	}{
		assessmentgate.SlotWaitCancelled: {
			verdictadding.SlotWaitDeadline,
			http.StatusServiceUnavailable,
		},
		assessmentgate.AssessmentCancelled: {
			verdictadding.AssessmentDeadline,
			http.StatusGatewayTimeout,
		},
		assessmentgate.Panicked: {
			verdictadding.UnassessedPage,
			http.StatusServiceUnavailable,
		},
	}
	for outcome, refusal := range refusals {
		fixture := newVerdictFixture()
		fixture.assessmentRunner.outcome = outcome
		ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
		defer cancelRoundTrip(nil)
		if outcome != assessmentgate.Panicked {
			fixture.assessmentRunner.during = func() {
				cancelRoundTrip(roundtripcancel.ErrHeadersDeadlinePassed)
			}
		}

		response := fixture.roundTrip(t, ctx, http.MethodGet)

		if response.status != refusal.status || response.headers.Get("Spam-Assessment") != "" ||
			fixture.observers.refusals[0] != refusal.reason {
			t.Errorf("outcome %v: response %d %v, refusals %v",
				outcome, response.status, response.headers, fixture.observers.refusals)
		}
	}
}

func TestAPageAssessedPastTheDeadlineGivesGatewayTimeout(t *testing.T) {
	for name, passDeadline := range map[string]func(*verdictFixture, context.CancelCauseFunc){
		"clock": func(fixture *verdictFixture, _ context.CancelCauseFunc) {
			fixture.clock.passTo(time.Second)
		},
		"cancellation": func(_ *verdictFixture, cancelRoundTrip context.CancelCauseFunc) {
			cancelRoundTrip(roundtripcancel.ErrHeadersDeadlinePassed)
		},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newVerdictFixture()
			ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
			fixture.assessmentRunner.during = func() { passDeadline(fixture, cancelRoundTrip) }

			response := fixture.roundTrip(t, ctx, http.MethodGet)

			if response.status != http.StatusGatewayTimeout ||
				response.headers.Get("Spam-Assessment") != "" ||
				fixture.observers.refusals[0] != verdictadding.AssessmentDeadline {
				t.Fatalf("response %d %v, refusals %v",
					response.status, response.headers, fixture.observers.refusals)
			}
		})
	}
}

func TestAPageWhoseClientLeavesDuringTheAssessmentGetsNoResponse(t *testing.T) {
	fixture := newVerdictFixture()
	fixture.assessmentRunner.outcome = assessmentgate.AssessmentCancelled
	ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
	fixture.assessmentRunner.during = func() { cancelRoundTrip(context.Canceled) }

	err := failureOf(fixture.roundTripper().RoundTrip(ctx, requestWith(t, http.MethodGet)))

	if !errors.Is(err, context.Canceled) || len(fixture.observers.refusals) != 0 {
		t.Fatalf("error %v, refusals %v", err, fixture.observers.refusals)
	}
}
