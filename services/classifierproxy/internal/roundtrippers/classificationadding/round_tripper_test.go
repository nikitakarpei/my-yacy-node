package classificationadding_test

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
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/classificationgate"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/proxiedrequest"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtripcancel"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippers/classificationadding"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
	pageclassificationhttpheader "github.com/nikitakarpei/yacy-rwi-node/pageclassification/httpheader"
)

const pageBody = "<html><title>Cheap pills</title><p>Buy cheap pills now</p></html>"

const pageByteCeiling = 1000

var (
	requestArrivedAt   = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	headersDeadline    = requestArrivedAt.Add(time.Second)
	pageClassification = pageclassification.Classification{
		pageclassification.Spam: {Score: 0.9, Threshold: 0.8, ModelVersion: "2026-09"},
		"adult":                 {Score: 0.1, Threshold: 0.75, ModelVersion: "2026-11"},
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

type fakeClassificationRunner struct {
	mutex           sync.Mutex
	classifiedPages [][]byte
	outcome         classificationgate.Outcome
	during          func()
}

func (a *fakeClassificationRunner) Classify(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	body []byte,
	_ http.Header,
) (pageclassification.Classification, classificationgate.Outcome) {
	a.mutex.Lock()
	a.classifiedPages = append(a.classifiedPages, body)
	a.mutex.Unlock()
	if a.during != nil {
		a.during()
	}
	return pageClassification, a.outcome
}

type observerRecord struct {
	mutex                  sync.Mutex
	refusals               []classificationadding.RefusalReason
	skippedClassifications []classificationadding.SkipReason
	bodyPrefixReadFailures []error
	classified             []pageclassification.Classification
	readingSlotWaits       []time.Duration
}

func (r *observerRecord) PageClassified(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	classification pageclassification.Classification,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.classified = append(r.classified, classification)
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

func (r *observerRecord) ClassificationSkipped(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason classificationadding.SkipReason,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.skippedClassifications = append(r.skippedClassifications, reason)
}

func (r *observerRecord) PageRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason classificationadding.RefusalReason,
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

type classificationFixture struct {
	upstream             *fakeUpstream
	classificationRunner *fakeClassificationRunner
	clock                *fakeClock
	observers            *observerRecord
	maxPagesReadAtOnce   int
}

func newClassificationFixture() *classificationFixture {
	return &classificationFixture{
		upstream:             &fakeUpstream{upstreamResponse: htmlUpstreamResponseWith(pageBody)},
		classificationRunner: &fakeClassificationRunner{},
		clock:                &fakeClock{now: requestArrivedAt},
		observers:            &observerRecord{},
		maxPagesReadAtOnce:   4,
	}
}

func (f *classificationFixture) roundTripper() *classificationadding.RoundTripper {
	return classificationadding.New(
		f.upstream,
		f.classificationRunner,
		classificationadding.Limits{
			PageByteCeiling:    pageByteCeiling,
			MaxPagesReadAtOnce: f.maxPagesReadAtOnce,
		},
		f.clock,
		classificationadding.Observers{f.observers},
	)
}

type readResponse struct {
	status  int
	headers http.Header
	body    string
}

func (f *classificationFixture) roundTrip(
	t *testing.T,
	ctx context.Context,
	method string,
) readResponse {
	t.Helper()
	response, err := f.roundTripper().RoundTrip(ctx, requestWith(t, method))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	return readResponseOf(t, response)
}

func (f *classificationFixture) roundTripInBackground(
	t *testing.T,
	roundTripper *classificationadding.RoundTripper,
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

func TestAnHTMLPageCarriesItsClassificationAndItsBody(t *testing.T) {
	fixture := newClassificationFixture()

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	wantClassification := pageclassificationhttpheader.ValueOf(pageClassification)
	if response.status != http.StatusOK ||
		response.headers.Get("Page-Classification") != wantClassification ||
		response.body != pageBody {
		t.Fatalf("response %d %v %q", response.status, response.headers, response.body)
	}
	if string(fixture.classificationRunner.classifiedPages[0]) != pageBody {
		t.Fatalf("classified body %q", fixture.classificationRunner.classifiedPages[0])
	}
}

func TestTheUpstreamIsAskedForTheAddressWithTheRequestHeaders(t *testing.T) {
	fixture := newClassificationFixture()

	fixture.roundTrip(t, t.Context(), http.MethodGet)

	request := fixture.upstream.requests[0]
	if request.Method != http.MethodGet || request.URL.String() != "http://site.example/page" ||
		request.Header.Get("User-Agent") != "crawler" {
		t.Fatalf("upstream request %s %s %v", request.Method, request.URL, request.Header)
	}
}

func TestTheClassificationOfTheOriginIsReplaced(t *testing.T) {
	fixture := newClassificationFixture()
	fixture.upstream.upstreamResponse.headers.Set("Page-Classification", "spam=?0")

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	if classificationValues := response.headers.Values(
		"Page-Classification",
	); len(
		classificationValues,
	) != 1 ||
		classificationValues[0] != pageclassificationhttpheader.ValueOf(pageClassification) {
		t.Fatalf("classifications %v", classificationValues)
	}
}

func TestAnUpstreamResponseWhoseClassificationIsSkippedIsReturnedWithoutAClassification(
	t *testing.T,
) {
	upstreamResponses := map[classificationadding.SkipReason]struct {
		method           string
		upstreamResponse upstreamResponse
	}{
		classificationadding.NotHTML: {http.MethodGet, upstreamResponse{
			status:  http.StatusOK,
			headers: http.Header{"Content-Type": {"image/png"}, "Page-Classification": {"spam=?0"}},
			body:    func() io.Reader { return strings.NewReader("png") },
		}},
		classificationadding.Not2xx: {
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
		classificationadding.HeadRequest: {http.MethodHead, htmlUpstreamResponseWith("png")},
	}
	for reason, skippedResponse := range upstreamResponses {
		t.Run(string(reason), func(t *testing.T) {
			fixture := newClassificationFixture()
			fixture.upstream.upstreamResponse = skippedResponse.upstreamResponse

			response := fixture.roundTrip(t, t.Context(), skippedResponse.method)

			if response.status != skippedResponse.upstreamResponse.status ||
				response.headers.Get("Page-Classification") != "" || response.body != "png" ||
				fixture.observers.skippedClassifications[0] != reason {
				t.Fatalf("response %d %v %q, reasons %v", response.status, response.headers,
					response.body, fixture.observers.skippedClassifications)
			}
		})
	}
}

func TestAGzipPageIsClassifiedDecodedAndReturnedAsTheOriginSentIt(t *testing.T) {
	fixture := newClassificationFixture()
	var encodedBody bytes.Buffer
	writer := gzip.NewWriter(&encodedBody)
	_, _ = writer.Write([]byte(pageBody))
	_ = writer.Close()
	fixture.upstream.upstreamResponse = htmlUpstreamResponseWith(encodedBody.String())
	fixture.upstream.upstreamResponse.headers.Set("Content-Encoding", "gzip")

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	if response.headers.Get("Page-Classification") == "" ||
		response.headers.Get("Content-Encoding") != "gzip" ||
		response.body != encodedBody.String() {
		t.Fatalf("response %v %q", response.headers, response.body)
	}
	if string(fixture.classificationRunner.classifiedPages[0]) != pageBody {
		t.Fatalf("classified body %q", fixture.classificationRunner.classifiedPages[0])
	}
}

func TestAPageThatCannotBeDecodedGivesBadGateway(t *testing.T) {
	for encoding, reason := range map[string]classificationadding.RefusalReason{
		"br":   classificationadding.UndecodableEncoding,
		"gzip": classificationadding.UndecodableBody,
	} {
		t.Run(encoding, func(t *testing.T) {
			fixture := newClassificationFixture()
			fixture.upstream.upstreamResponse = htmlUpstreamResponseWith("zipped")
			fixture.upstream.upstreamResponse.headers.Set("Content-Encoding", encoding)

			response := fixture.roundTrip(t, t.Context(), http.MethodGet)

			if response.status != http.StatusBadGateway || fixture.observers.refusals[0] != reason {
				t.Fatalf("status %d, refusals %v", response.status, fixture.observers.refusals)
			}
		})
	}
}

func TestAPageOverTheByteCeilingIsClassifiedOnItsFirstBytesAndReturnedWhole(t *testing.T) {
	fixture := newClassificationFixture()
	body := "<p>" + strings.Repeat("x", pageByteCeiling) + "</p>"
	fixture.upstream.upstreamResponse = htmlUpstreamResponseWith(body)
	fixture.upstream.upstreamResponse.headers.Set("Content-Length", "1007")

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	if response.headers.Get("Content-Length") != "1007" || response.body != body {
		t.Fatalf("response %v, %d bytes", response.headers, len(response.body))
	}
	if classifiedBytes := len(
		fixture.classificationRunner.classifiedPages[0],
	); classifiedBytes != pageByteCeiling {
		t.Fatalf("classified %d bytes", classifiedBytes)
	}
}

func TestAWholePageOfUnknownLengthGetsItsLength(t *testing.T) {
	fixture := newClassificationFixture()

	response := fixture.roundTrip(t, t.Context(), http.MethodGet)

	if response.headers.Get("Content-Length") != "65" {
		t.Fatalf("content length %q", response.headers.Get("Content-Length"))
	}
}

func TestAFailedUpstreamIsPassedOnUnchanged(t *testing.T) {
	fixture := newClassificationFixture()
	fixture.upstream.failure = errEgressDown

	err := failureOf(fixture.roundTripper().RoundTrip(t.Context(), requestWith(t, http.MethodGet)))

	if !errors.Is(err, errEgressDown) {
		t.Fatalf("error %v", err)
	}
}

func TestAPageWhoseBodyPrefixIsCutAtTheDeadlineGivesGatewayTimeout(t *testing.T) {
	fixture := newClassificationFixture()
	ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
	fixture.upstream.upstreamResponse.body = func() io.Reader {
		return cancellingBody{
			cancelReading: cancelRoundTrip,
			cause:         roundtripcancel.ErrHeadersDeadlinePassed,
		}
	}

	response := fixture.roundTrip(t, ctx, http.MethodGet)

	if response.status != http.StatusGatewayTimeout ||
		fixture.observers.refusals[0] != classificationadding.BodyPrefixReadDeadline {
		t.Fatalf("status %d, refusals %v", response.status, fixture.observers.refusals)
	}
}

func TestAPageWhoseBodyPrefixFailsToReadGivesBadGateway(t *testing.T) {
	fixture := newClassificationFixture()
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
	fixture := newClassificationFixture()
	ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
	fixture.upstream.upstreamResponse.body = func() io.Reader {
		return cancellingBody{cancelReading: cancelRoundTrip, cause: context.Canceled}
	}

	err := failureOf(fixture.roundTripper().RoundTrip(ctx, requestWith(t, http.MethodGet)))

	if !errors.Is(err, context.Canceled) || len(fixture.observers.refusals) != 0 {
		t.Fatalf("error %v, refusals %v", err, fixture.observers.refusals)
	}
}

func TestAPageWaitingForAReadingSlotIsClassifiedOnceTheSlotFrees(t *testing.T) {
	fixture := newClassificationFixture()
	fixture.maxPagesReadAtOnce = 1
	started, released := make(chan struct{}), make(chan struct{})
	fixture.classificationRunner.during = heldOnFirstClassification(started, released)
	roundTripper := fixture.roundTripper()
	heldResponses := fixture.roundTripInBackground(t, roundTripper, t.Context())
	<-started
	waitingResponses := fixture.roundTripInBackground(t, roundTripper, t.Context())

	close(released)

	if waitingResponse := <-waitingResponses; waitingResponse.headers.Get(
		"Page-Classification",
	) == "" {
		t.Fatalf("waiting response %d %v", waitingResponse.status, waitingResponse.headers)
	}
	if heldResponse := <-heldResponses; heldResponse.headers.Get("Page-Classification") == "" {
		t.Fatalf("held response %v", heldResponse.headers)
	}
	if slotWaits := fixture.observers.readingSlotWaits; len(slotWaits) != 2 {
		t.Fatalf("reading slot waits %v, want one for each page", slotWaits)
	}
}

func TestAPageWithoutAReadingSlotAtTheDeadlineGivesServiceUnavailable(t *testing.T) {
	fixture := newClassificationFixture()
	fixture.maxPagesReadAtOnce = 1
	started, released := make(chan struct{}), make(chan struct{})
	fixture.classificationRunner.during = heldOnFirstClassification(started, released)
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
	if heldResponse := <-heldResponses; heldResponse.headers.Get("Page-Classification") == "" {
		t.Fatalf("held response %v", heldResponse.headers)
	}
	if refusals := fixture.observers.refusals; len(refusals) != 1 ||
		refusals[0] != classificationadding.SlotWaitDeadline {
		t.Fatalf("refusals %v", refusals)
	}
}

func heldOnFirstClassification(started chan<- struct{}, released <-chan struct{}) func() {
	var classifications atomic.Int32
	return func() {
		if classifications.Add(1) == 1 {
			close(started)
			<-released
		}
	}
}

func TestAPageThatWasNotClassifiedIsRefused(t *testing.T) {
	refusals := map[classificationgate.Outcome]struct {
		reason classificationadding.RefusalReason
		status int
	}{
		classificationgate.SlotWaitCancelled: {
			classificationadding.SlotWaitDeadline,
			http.StatusServiceUnavailable,
		},
		classificationgate.ClassificationCancelled: {
			classificationadding.ClassificationDeadline,
			http.StatusGatewayTimeout,
		},
		classificationgate.Panicked: {
			classificationadding.UnclassifiedPage,
			http.StatusServiceUnavailable,
		},
	}
	for outcome, refusal := range refusals {
		fixture := newClassificationFixture()
		fixture.classificationRunner.outcome = outcome
		ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
		defer cancelRoundTrip(nil)
		if outcome != classificationgate.Panicked {
			fixture.classificationRunner.during = func() {
				cancelRoundTrip(roundtripcancel.ErrHeadersDeadlinePassed)
			}
		}

		response := fixture.roundTrip(t, ctx, http.MethodGet)

		if response.status != refusal.status || response.headers.Get("Page-Classification") != "" ||
			fixture.observers.refusals[0] != refusal.reason {
			t.Errorf("outcome %v: response %d %v, refusals %v",
				outcome, response.status, response.headers, fixture.observers.refusals)
		}
	}
}

func TestAPageClassifiedPastTheDeadlineGivesGatewayTimeout(t *testing.T) {
	for name, passDeadline := range map[string]func(*classificationFixture, context.CancelCauseFunc){
		"clock": func(fixture *classificationFixture, _ context.CancelCauseFunc) {
			fixture.clock.passTo(time.Second)
		},
		"cancellation": func(_ *classificationFixture, cancelRoundTrip context.CancelCauseFunc) {
			cancelRoundTrip(roundtripcancel.ErrHeadersDeadlinePassed)
		},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newClassificationFixture()
			ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
			fixture.classificationRunner.during = func() { passDeadline(fixture, cancelRoundTrip) }

			response := fixture.roundTrip(t, ctx, http.MethodGet)

			if response.status != http.StatusGatewayTimeout ||
				response.headers.Get("Page-Classification") != "" ||
				fixture.observers.refusals[0] != classificationadding.ClassificationDeadline {
				t.Fatalf("response %d %v, refusals %v",
					response.status, response.headers, fixture.observers.refusals)
			}
		})
	}
}

func TestAPageWhoseClientLeavesDuringTheClassificationGetsNoResponse(t *testing.T) {
	fixture := newClassificationFixture()
	fixture.classificationRunner.outcome = classificationgate.ClassificationCancelled
	ctx, cancelRoundTrip := context.WithCancelCause(t.Context())
	fixture.classificationRunner.during = func() { cancelRoundTrip(context.Canceled) }

	err := failureOf(fixture.roundTripper().RoundTrip(ctx, requestWith(t, http.MethodGet)))

	if !errors.Is(err, context.Canceled) || len(fixture.observers.refusals) != 0 {
		t.Fatalf("error %v, refusals %v", err, fixture.observers.refusals)
	}
}
