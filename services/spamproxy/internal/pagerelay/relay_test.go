package pagerelay_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/pagerelay"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/replydeadlines"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
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
	testLimits    = pagerelay.Limits{
		PageByteCeiling:        pageByteCeiling,
		MaxPagesAssessedAtOnce: 4,
		ReplyTimeouts: replydeadlines.Timeouts{
			ResponseHeader:   time.Second,
			AssessmentBudget: 500 * time.Millisecond,
		},
		RelayIdleTimeout: 500 * time.Millisecond,
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
	assessed      bool
	during        func()
}

func newFakeAssessor() *fakeAssessor {
	return &fakeAssessor{assessed: true}
}

func (a *fakeAssessor) AssessmentFrom(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
	_ time.Time,
) (spamassessment.Assessment, bool) {
	a.mutex.Lock()
	a.assessedPages = append(a.assessedPages, assessedBody{body: body, headers: responseHeaders})
	a.mutex.Unlock()
	if a.during != nil {
		a.during()
	}
	return spamAssessment, a.assessed
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
	mutex    sync.Mutex
	refusals []pagerelay.RefusalReason
	nonPages []pagerelay.NonPageReason
	failures []error
	assessed []spamassessment.Assessment
}

func (r *observerRecord) PageAssessed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
	_ time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.assessed = append(r.assessed, assessment)
}

func (r *observerRecord) NonPageRelayed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason pagerelay.NonPageReason,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.nonPages = append(r.nonPages, reason)
}

func (r *observerRecord) AnswerRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason pagerelay.RefusalReason,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.refusals = append(r.refusals, reason)
}

func (r *observerRecord) RelayFailed(_ context.Context, _ canonicalurl.CanonicalURL, cause error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failures = append(r.failures, cause)
}

type relayFixture struct {
	egress    *fakeEgress
	assessor  *fakeAssessor
	clock     *fakeClock
	observers *observerRecord
	limits    pagerelay.Limits
}

func newRelayFixture() *relayFixture {
	return &relayFixture{
		egress:    &fakeEgress{answer: htmlAnswerWith(pageBody)},
		assessor:  newFakeAssessor(),
		clock:     newFakeClock(),
		observers: &observerRecord{},
		limits:    testLimits,
	}
}

func (f *relayFixture) relay() *pagerelay.Relay {
	return pagerelay.New(f.egress, f.assessor, pagerelay.Observers{f.observers}, f.limits, f.clock)
}

func (f *relayFixture) replyTo(t *testing.T, method string, requestHeaders http.Header) *fakeReply {
	t.Helper()
	reply := &fakeReply{}
	f.relay().ReplyTo(t.Context(), method, pageAddress(t), requestHeaders, reply)
	return reply
}

func (f *relayFixture) replyInBackgroundTo(t *testing.T, relay *pagerelay.Relay) <-chan *fakeReply {
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
		reply.headers.Get("Spam-Assessment") != spamAssessment.HeaderValue() ||
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
		verdicts[0] != spamAssessment.HeaderValue() {
		t.Fatalf("verdicts %v", verdicts)
	}
}

func TestAnAnswerThatIsNotAPageIsRelayedWithoutAVerdict(t *testing.T) {
	answers := map[pagerelay.NonPageReason]struct {
		method string
		answer originAnswer
	}{
		pagerelay.NotHTML: {http.MethodGet, originAnswer{
			status:  http.StatusOK,
			headers: http.Header{"Content-Type": {"image/png"}, "Spam-Assessment": {"clean"}},
			body:    func(context.Context) io.Reader { return strings.NewReader("png") },
		}},
		pagerelay.Not2xx: {
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
		pagerelay.HeadRequest: {http.MethodHead, htmlAnswerWith("png")},
	}
	for reason, nonPage := range answers {
		t.Run(string(reason), func(t *testing.T) {
			fixture := newRelayFixture()
			fixture.egress.answer = nonPage.answer

			reply := fixture.replyTo(t, nonPage.method, http.Header{})

			if reply.status != nonPage.answer.status ||
				reply.headers.Get("Spam-Assessment") != "" ||
				reply.String() != "png" ||
				fixture.observers.nonPages[0] != reason {
				t.Fatalf(
					"reply %d %v %q, reasons %v",
					reply.status,
					reply.headers,
					reply.String(),
					fixture.observers.nonPages,
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
	for encoding, reason := range map[string]pagerelay.RefusalReason{
		"br":   pagerelay.UndecodableEncoding,
		"gzip": pagerelay.UndecodableBody,
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
		!errors.Is(fixture.observers.failures[0], errEgressDown) {
		t.Fatalf("status %d, failures %v", reply.status, fixture.observers.failures)
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
		fixture.observers.refusals[0] != pagerelay.PageReadTimeout {
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
		!errors.Is(fixture.observers.failures[0], io.ErrUnexpectedEOF) {
		t.Fatalf("status %d, failures %v", reply.status, fixture.observers.failures)
	}
}

func TestAnAnswerThatIsNotAPageCutShortEndsWhereTheEgressStopped(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer = originAnswer{
		status:  http.StatusOK,
		headers: http.Header{"Content-Type": {"image/png"}},
		body: func(context.Context) io.Reader {
			return io.MultiReader(strings.NewReader("png"), failingBody{io.ErrUnexpectedEOF})
		},
	}

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusOK || reply.String() != "png" || !reply.wasCutShort {
		t.Fatalf("reply %d %q, cut short %v", reply.status, reply.String(), reply.wasCutShort)
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

func TestAPagePastTheMaxPagesAssessedAtOnceGivesServiceUnavailable(t *testing.T) {
	fixture := newRelayFixture()
	fixture.limits.MaxPagesAssessedAtOnce = 1
	started, released := make(chan struct{}), make(chan struct{})
	fixture.assessor.during = func() {
		close(started)
		<-released
	}
	relay := fixture.relay()
	heldReplies := fixture.replyInBackgroundTo(t, relay)
	<-started

	passedReply := &fakeReply{}
	relay.ReplyTo(t.Context(), http.MethodGet, pageAddress(t), http.Header{}, passedReply)
	close(released)

	if passedReply.status != http.StatusServiceUnavailable ||
		passedReply.headers.Get("Retry-After") != "1" {
		t.Fatalf("passed reply %d %v", passedReply.status, passedReply.headers)
	}
	if heldReply := <-heldReplies; heldReply.headers.Get("Spam-Assessment") == "" {
		t.Fatalf("held reply %v", heldReply.headers)
	}
}

func TestAnAnswerThatStartsAfterReadingEndsGivesGatewayTimeout(t *testing.T) {
	fixture := newRelayFixture()
	fixture.egress.answer.before = func() { fixture.clock.passTo(600 * time.Millisecond) }

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusGatewayTimeout ||
		fixture.observers.refusals[0] != pagerelay.PageReadTimeout {
		t.Fatalf("status %d, refusals %v", reply.status, fixture.observers.refusals)
	}
}

func TestAPageLeftUnassessedGivesServiceUnavailable(t *testing.T) {
	fixture := newRelayFixture()
	fixture.assessor.assessed = false

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusServiceUnavailable || reply.headers.Get("Retry-After") != "1" ||
		reply.headers.Get("Spam-Assessment") != "" {
		t.Fatalf("reply %d %v", reply.status, reply.headers)
	}
}

func TestAPageAssessedPastTheResponseHeaderDeadlineGivesGatewayTimeout(t *testing.T) {
	fixture := newRelayFixture()
	fixture.assessor.during = func() { fixture.clock.passTo(time.Second) }

	reply := fixture.replyTo(t, http.MethodGet, http.Header{})

	if reply.status != http.StatusGatewayTimeout || reply.headers.Get("Spam-Assessment") != "" {
		t.Fatalf("reply %d %v", reply.status, reply.headers)
	}
}

func TestAWaitPreferenceTooShortForReadingGivesGatewayTimeoutWithoutAskingTheEgress(t *testing.T) {
	fixture := newRelayFixture()

	reply := fixture.replyTo(t, http.MethodGet, http.Header{"Prefer": {"wait=0"}})

	if reply.status != http.StatusGatewayTimeout || len(fixture.egress.requests) != 0 {
		t.Fatalf("status %d, egress requests %d", reply.status, len(fixture.egress.requests))
	}
}
