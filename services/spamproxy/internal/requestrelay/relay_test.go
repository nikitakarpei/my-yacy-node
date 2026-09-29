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

type originAnswer struct {
	status  int
	headers http.Header
	body    func(ctx context.Context) io.Reader
	before  func()
}

func htmlAnswerWith(body string) originAnswer {
	return originAnswer{
		status:  http.StatusOK,
		headers: http.Header{"Content-Type": {"text/html; charset=utf-8"}},
		body:    func(context.Context) io.Reader { return strings.NewReader(body) },
	}
}

type fakeEgress struct {
	mutex    sync.Mutex
	answer   originAnswer
	requests []*http.Request
	failure  error
}

func (e *fakeEgress) RoundTrip(request *http.Request) (*http.Response, error) {
	e.mutex.Lock()
	e.requests = append(e.requests, request)
	e.mutex.Unlock()
	if e.failure != nil {
		return nil, e.failure
	}
	if e.answer.before != nil {
		e.answer.before()
	}
	if e.answer.body == nil {
		<-request.Context().Done()
		return nil, fmt.Errorf("silent egress: %w", request.Context().Err())
	}
	return &http.Response{
		StatusCode: e.answer.status,
		Header:     e.answer.headers,
		Body:       io.NopCloser(e.answer.body(request.Context())),
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

type fakeAssessor struct {
	mutex         sync.Mutex
	assessedPages []assessedBody
	outcome       assessmentgate.Outcome
	during        func()
}

func (a *fakeAssessor) Assess(
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

type fakeReply struct {
	status  int
	headers http.Header
	bytes.Buffer
	wasCutShort bool
}

func (r *fakeReply) SendHead(status int, headers http.Header) {
	r.status, r.headers = status, headers
}

func (r *fakeReply) CutShort() { r.wasCutShort = true }

type observerRecord struct {
	mutex              sync.Mutex
	refusals           []requestrelay.RefusalReason
	skippedAssessments []requestrelay.SkipReason
	readingFailures    []error
	cutShortCauses     []error
	departedClients    int
	sentHeaders        []sentHeaders
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

func (r *observerRecord) AnswerReadingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.readingFailures = append(r.readingFailures, cause)
}

func (r *observerRecord) ReplyCutShort(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.cutShortCauses = append(r.cutShortCauses, cause)
}

func (r *observerRecord) ClientLeft(context.Context, canonicalurl.CanonicalURL) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.departedClients++
}

type sentHeaders struct {
	replyKind    requestrelay.ReplyKind
	headersDelay time.Duration
}

func (r *observerRecord) HeadersSent(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	replyKind requestrelay.ReplyKind,
	headersDelay time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.sentHeaders = append(r.sentHeaders, sentHeaders{replyKind, headersDelay})
}

type relayFixture struct {
	egress    *fakeEgress
	assessor  *fakeAssessor
	clock     *fakeClock
	observers *observerRecord
	limits    requestrelay.Limits
}

func newRelayFixture() *relayFixture {
	return &relayFixture{
		egress:    &fakeEgress{answer: htmlAnswerWith(pageBody)},
		assessor:  &fakeAssessor{},
		clock:     newFakeClock(),
		observers: &observerRecord{},
		limits:    testLimits,
	}
}

func (f *relayFixture) relay() *requestrelay.Relayer {
	return requestrelay.New(
		f.egress,
		f.assessor,
		requestrelay.Observers{f.observers},
		f.limits,
		f.clock,
	)
}

func (f *relayFixture) replyTo(t *testing.T, method string, requestHeaders http.Header) *fakeReply {
	t.Helper()
	reply := &fakeReply{}
	f.relay().ReplyTo(t.Context(), method, pageAddress(t), requestHeaders, reply)
	return reply
}

func (f *relayFixture) replyInBackgroundTo(
	t *testing.T,
	relay *requestrelay.Relayer,
) <-chan *fakeReply {
	t.Helper()
	replies := make(chan *fakeReply, 1)
	address := pageAddress(t)
	go func() {
		reply := &fakeReply{}
		relay.ReplyTo(context.Background(), http.MethodGet, address, http.Header{}, reply)
		replies <- reply
	}()
	return replies
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
	fixture := newRelayFixture()

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusOK ||
		reply.headers.Get("Spam-Assessment") != spamassessmenthttpheader.ValueOf(spamAssessment) ||
		reply.String() != pageBody {
		t.Fatalf("reply %d %v %q", reply.status, reply.headers, reply.String())
	}
	if string(fixture.assessor.assessedPages[0].body) != pageBody {
		t.Fatalf("assessed body %q", fixture.assessor.assessedPages[0].body)
	}
}

func TestTheEgressIsAskedForTheAddressWithTheForwardedHeaders(t *testing.T) {
	fixture := newRelayFixture()

	fixture.replyTo(
		t,
		http.MethodGet,
		http.Header{"User-Agent": {"crawler"}, "Cookie": {"session=1"}},
	)

	request := fixture.egress.requests[0]
	if request.URL.String() != "http://site.example/page" ||
		request.Header.Get("User-Agent") != "crawler" ||
		request.Header.Get("Cookie") != "" {
		t.Fatalf("egress request %s %v", request.URL, request.Header)
	}
}

func TestTheVerdictOfTheOriginIsReplaced(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer.headers.Set("Spam-Assessment", "clean")

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if verdicts := reply.headers.Values("Spam-Assessment"); len(verdicts) != 1 ||
		verdicts[0] != spamassessmenthttpheader.ValueOf(spamAssessment) {
		t.Fatalf("verdicts %v", verdicts)
	}
}

func TestAnAnswerWhoseAssessmentIsSkippedIsRelayedWithoutAVerdict(t *testing.T) {
	answers := map[requestrelay.SkipReason]struct {
		method string
		answer originAnswer
	}{
		requestrelay.NotHTML: {http.MethodGet, originAnswer{
			status:  http.StatusOK,
			headers: http.Header{"Content-Type": {"image/png"}, "Spam-Assessment": {"clean"}},
			body:    func(context.Context) io.Reader { return strings.NewReader("png") },
		}},
		requestrelay.Not2xx: {
			http.MethodGet,
			originAnswer{
				status: http.StatusFound,
				headers: http.Header{
					"Content-Type": {"text/html"},
					"Location":     {"http://other.example/"},
				},
				body: func(context.Context) io.Reader { return strings.NewReader("png") },
			},
		},
		requestrelay.HeadRequest: {http.MethodHead, htmlAnswerWith("png")},
	}
	for reason, skippedAnswer := range answers {
		t.Run(string(reason), func(t *testing.T) {
			fixture := newRelayFixture()
			fixture.egress.answer = skippedAnswer.answer

			reply := fixture.replyTo(t, skippedAnswer.method, http.Header{})

			if reply.status != skippedAnswer.answer.status ||
				reply.headers.Get("Spam-Assessment") != "" ||
				reply.String() != "png" ||
				fixture.observers.skippedAssessments[0] != reason {
				t.Fatalf(
					"reply %d %v %q, reasons %v",
					reply.status,
					reply.headers,
					reply.String(),
					fixture.observers.skippedAssessments,
				)
			}
		})
	}
}

func TestAGzipPageIsAssessedDecodedAndRelayedAsTheOriginSentIt(t *testing.T) {
	fixture := newRelayFixture()
	var encodedBody bytes.Buffer
	writer := gzip.NewWriter(&encodedBody)
	_, _ = writer.Write([]byte(pageBody))
	_ = writer.Close()
	fixture.egress.answer = htmlAnswerWith(encodedBody.String())
	fixture.egress.answer.headers.Set("Content-Encoding", "gzip")

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.headers.Get("Spam-Assessment") == "" ||
		reply.headers.Get("Content-Encoding") != "gzip" ||
		reply.String() != encodedBody.String() {
		t.Fatalf("reply %v %q", reply.headers, reply.String())
	}
	if string(fixture.assessor.assessedPages[0].body) != pageBody {
		t.Fatalf("assessed body %q", fixture.assessor.assessedPages[0].body)
	}
}

func TestAPageThatCannotBeDecodedGivesBadGateway(t *testing.T) {
	for encoding, reason := range map[string]requestrelay.RefusalReason{
		"br":   requestrelay.UndecodableEncoding,
		"gzip": requestrelay.UndecodableBody,
	} {
		t.Run(encoding, func(t *testing.T) {
			fixture := newRelayFixture()
			fixture.egress.answer = htmlAnswerWith("zipped")
			fixture.egress.answer.headers.Set("Content-Encoding", encoding)

			reply := fixture.replyTo(t, http.MethodGet, http.Header{})

			if reply.status != http.StatusBadGateway || fixture.observers.refusals[0] != reason {
				t.Fatalf("status %d, refusals %v", reply.status, fixture.observers.refusals)
			}
		})
	}
}

func TestAPageOverTheByteCeilingIsAssessedOnItsFirstBytesAndRelayedWhole(t *testing.T) {
	fixture := newRelayFixture()
	body := "<p>" + strings.Repeat("x", pageByteCeiling) + "</p>"
	fixture.egress.answer = htmlAnswerWith(body)
	fixture.egress.answer.headers.Set("Content-Length", "1007")

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.headers.Get("Content-Length") != "1007" || reply.String() != body {
		t.Fatalf("reply %v, %d bytes", reply.headers, reply.Len())
	}
	if assessedBytes := len(
		fixture.assessor.assessedPages[0].body,
	); assessedBytes != pageByteCeiling {
		t.Fatalf("assessed %d bytes", assessedBytes)
	}
}

func TestAPageOverTheByteCeilingAssessedPastTheEndOfReadingIsRelayedWhole(t *testing.T) {
	fixture := newRelayFixture()
	body := "<p>" + strings.Repeat("x", pageByteCeiling) + "</p>"
	fixture.egress.answer = htmlAnswerWith(body)
	fixture.egress.answer.body = func(ctx context.Context) io.Reader {
		return cancellableBody{ctx: ctx, body: strings.NewReader(body)}
	}
	fixture.assessor.during = func() { (<-fixture.clock.expirations)() }

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.wasCutShort || reply.String() != body {
		t.Fatalf("cut short %v, %d bytes", reply.wasCutShort, reply.Len())
	}
}

func TestAWholePageOfUnknownLengthGetsItsLength(t *testing.T) {
	fixture := newRelayFixture()

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.headers.Get("Content-Length") != "65" {
		t.Fatalf("content length %q", reply.headers.Get("Content-Length"))
	}
}

func TestAFailedEgressGivesBadGateway(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.failure = errEgressDown

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusBadGateway ||
		!errors.Is(fixture.observers.readingFailures[0], errEgressDown) {
		t.Fatalf("status %d, failures %v", reply.status, fixture.observers.readingFailures)
	}
}

func TestASilentEgressGivesGatewayTimeoutWhenReadingEnds(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer.body = nil
	replies := fixture.replyInBackgroundTo(t, fixture.relay())

	(<-fixture.clock.expirations)()

	if reply := <-replies; reply.status != http.StatusGatewayTimeout {
		t.Fatalf("status %d", reply.status)
	}
}

func TestAPageWhoseBodyStallsGivesGatewayTimeoutWhenReadingEnds(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer.body = func(ctx context.Context) io.Reader { return stallingBody{ctx} }
	replies := fixture.replyInBackgroundTo(t, fixture.relay())

	(<-fixture.clock.expirations)()

	if reply := <-replies; reply.status != http.StatusGatewayTimeout ||
		fixture.observers.refusals[0] != requestrelay.PageReadDeadline {
		t.Fatalf("status %d, refusals %v", reply.status, fixture.observers.refusals)
	}
}

func TestAPageCutShortByTheEgressGivesBadGateway(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer.body = func(context.Context) io.Reader {
		return io.MultiReader(strings.NewReader("<p>short</p>"), failingBody{io.ErrUnexpectedEOF})
	}

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusBadGateway ||
		!errors.Is(fixture.observers.readingFailures[0], io.ErrUnexpectedEOF) {
		t.Fatalf("status %d, failures %v", reply.status, fixture.observers.readingFailures)
	}
}

func TestAnAnswerWhoseAssessmentIsSkippedCutShortEndsWhereTheEgressStopped(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer = originAnswer{
		status:  http.StatusOK,
		headers: http.Header{"Content-Type": {"image/png"}},
		body: func(context.Context) io.Reader {
			return io.MultiReader(strings.NewReader("png"), failingBody{io.ErrUnexpectedEOF})
		},
	}

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusOK || reply.String() != "png" || !reply.wasCutShort ||
		!errors.Is(fixture.observers.cutShortCauses[0], io.ErrUnexpectedEOF) {
		t.Fatalf("reply %d %q, cut short %v", reply.status, reply.String(), reply.wasCutShort)
	}
}

func TestAClientThatLeavesBeforeTheAnswerIsReportedAsLeft(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer.body = nil
	clientCtx, leave := context.WithCancel(t.Context())
	leave()

	fixture.relay().ReplyTo(clientCtx, http.MethodGet, pageAddress(t), http.Header{}, &fakeReply{})

	if fixture.observers.departedClients != 1 || len(fixture.observers.readingFailures) != 0 {
		t.Fatalf("departed %d, reading failures %v",
			fixture.observers.departedClients, fixture.observers.readingFailures)
	}
}

func TestAClientThatLeavesDuringTheBodyIsReportedAsLeft(t *testing.T) {
	fixture := newRelayFixture()
	clientCtx, leave := context.WithCancel(t.Context())
	fixture.egress.answer = originAnswer{
		status:  http.StatusOK,
		headers: http.Header{"Content-Type": {"image/png"}},
		body: func(context.Context) io.Reader {
			leave()
			return io.MultiReader(strings.NewReader("png"), failingBody{io.ErrUnexpectedEOF})
		},
	}
	reply := &fakeReply{}

	fixture.relay().ReplyTo(clientCtx, http.MethodGet, pageAddress(t), http.Header{}, reply)

	if !reply.wasCutShort || fixture.observers.departedClients != 1 ||
		len(fixture.observers.cutShortCauses) != 0 {
		t.Fatalf("cut short %v, departed %d, cut short causes %v", reply.wasCutShort,
			fixture.observers.departedClients, fixture.observers.cutShortCauses)
	}
}

func TestTheTimeUntilTheHeadersIsReportedWithTheKindOfReply(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer.before = func() { fixture.clock.passTo(300 * time.Millisecond) }

	fixture.replyTo(t, http.MethodGet, http.Header{})

	want := []sentHeaders{{requestrelay.AssessedReply, 300 * time.Millisecond}}
	if !slices.Equal(fixture.observers.sentHeaders, want) {
		t.Fatalf("sent headers %v, want %v", fixture.observers.sentHeaders, want)
	}
}

func TestAPageWhoseRestStallsIsCutShortAfterTheIdleTimeout(t *testing.T) {
	fixture := newRelayFixture()
	fixture.limits.PageByteCeiling = 8
	fixture.egress.answer.body = func(ctx context.Context) io.Reader {
		return io.MultiReader(strings.NewReader(pageBody), stallingBody{ctx})
	}
	replies := fixture.replyInBackgroundTo(t, fixture.relay())

	<-fixture.clock.expirations
	<-fixture.clock.expirations
	(<-fixture.clock.expirations)()

	if reply := <-replies; reply.status != http.StatusOK || reply.String() != pageBody ||
		!reply.wasCutShort {
		t.Fatalf("reply %d %q, cut short %v", reply.status, reply.String(), reply.wasCutShort)
	}
}

func TestAPageWaitingForAReadingSlotIsRelayedOnceTheSlotFrees(t *testing.T) {
	fixture := newRelayFixture()
	fixture.limits.MaxPagesReadAtOnce = 1
	started, released := make(chan struct{}), make(chan struct{})
	fixture.assessor.during = heldOnFirstAssessment(started, released)
	relay := fixture.relay()
	heldReplies := fixture.replyInBackgroundTo(t, relay)
	<-started
	waitingReplies := fixture.replyInBackgroundTo(t, relay)
	<-fixture.clock.expirations
	<-fixture.clock.expirations

	close(released)

	if waitingReply := <-waitingReplies; waitingReply.status != http.StatusOK ||
		waitingReply.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("waiting reply %d %v", waitingReply.status, waitingReply.headers)
	}
	if heldReply := <-heldReplies; heldReply.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("held reply %v", heldReply.headers)
	}
}

func TestAPageWithoutAReadingSlotWhenTheHeadersAreDueGivesServiceUnavailable(t *testing.T) {
	fixture := newRelayFixture()
	fixture.limits.MaxPagesReadAtOnce = 1
	started, released := make(chan struct{}), make(chan struct{})
	fixture.assessor.during = heldOnFirstAssessment(started, released)
	relay := fixture.relay()
	heldReplies := fixture.replyInBackgroundTo(t, relay)
	<-started
	waitingReplies := fixture.replyInBackgroundTo(t, relay)
	<-fixture.clock.expirations

	(<-fixture.clock.expirations)()
	waitingReply := <-waitingReplies
	close(released)

	if waitingReply.status != http.StatusServiceUnavailable ||
		waitingReply.headers.Get("Retry-After") != "1" {
		t.Fatalf("waiting reply %d %v", waitingReply.status, waitingReply.headers)
	}
	if heldReply := <-heldReplies; heldReply.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("held reply %v", heldReply.headers)
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

func TestAnAnswerThatStartsAfterTheHeadersAreDueGivesGatewayTimeout(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer.before = func() { fixture.clock.passTo(1100 * time.Millisecond) }

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusGatewayTimeout ||
		fixture.observers.refusals[0] != requestrelay.PageReadDeadline {
		t.Fatalf("status %d, refusals %v", reply.status, fixture.observers.refusals)
	}
}

func TestAPageThatTheAssessorDidNotAssessIsRefused(t *testing.T) {
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
		fixture := newRelayFixture()
		fixture.assessor.outcome = outcome

		reply := fixture.replyTo(t, http.MethodGet, http.Header{})

		if reply.status != refusal.status || reply.headers.Get("Spam-Assessment") != "" ||
			fixture.observers.refusals[0] != refusal.reason {
			t.Errorf("outcome %v: reply %d %v, refusals %v",
				outcome, reply.status, reply.headers, fixture.observers.refusals)
		}
	}
}

func TestAPageAssessedPastTheResponseHeaderDeadlineGivesGatewayTimeout(t *testing.T) {
	fixture := newRelayFixture()
	fixture.assessor.during = func() { fixture.clock.passTo(time.Second) }

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusGatewayTimeout || reply.headers.Get("Spam-Assessment") != "" ||
		fixture.observers.refusals[0] != requestrelay.AssessmentDeadline {
		t.Fatalf(
			"reply %d %v, refusals %v",
			reply.status,
			reply.headers,
			fixture.observers.refusals,
		)
	}
}

func TestAWaitPreferenceOfZeroGivesGatewayTimeoutWithoutAskingTheEgress(t *testing.T) {
	fixture := newRelayFixture()

	reply := fixture.replyTo(t, http.MethodGet, http.Header{"Prefer": {"wait=0"}})

	if reply.status != http.StatusGatewayTimeout || len(fixture.egress.requests) != 0 {
		t.Fatalf("status %d, egress requests %d", reply.status, len(fixture.egress.requests))
	}
}
