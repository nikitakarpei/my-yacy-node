// Package verdictadding adds a spam verdict to each page that the origin sends,
// and relays every other upstream response as it is.
package verdictadding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	spamassessmenthttpheader "github.com/nikitakarpei/yacy-rwi-node/spamassessment/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxiedrequest"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtripcancel"
)

const retryAfterSeconds = "1"

type Upstream interface {
	RoundTrip(request *http.Request) (*http.Response, error)
}

type AssessmentRunner interface {
	Assess(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		body []byte,
		responseHeaders http.Header,
	) (spamassessment.Assessment, assessmentgate.Outcome)
}

type Clock interface {
	Now() time.Time
}

type Limits struct {
	PageByteCeiling    int
	MaxPagesReadAtOnce int
}

type RoundTripper struct {
	upstream         Upstream
	assessmentRunner AssessmentRunner
	observers        Observers
	clock            Clock
	pageByteCeiling  int
	readingSlots     readingSlots
}

func New(
	upstream Upstream,
	assessmentRunner AssessmentRunner,
	limits Limits,
	clock Clock,
	observers Observers,
) *RoundTripper {
	return &RoundTripper{
		upstream:         upstream,
		assessmentRunner: assessmentRunner,
		observers:        observers,
		clock:            clock,
		pageByteCeiling:  limits.PageByteCeiling,
		readingSlots:     make(readingSlots, limits.MaxPagesReadAtOnce),
	}
}

func (r *RoundTripper) RoundTrip(
	ctx context.Context,
	request proxiedrequest.Request,
) (*http.Response, error) {
	upstreamResponse, err := r.upstream.RoundTrip(upstreamRequestFor(ctx, request))
	if err != nil {
		return nil, err
	}
	if reason, skipped := skipReasonOf(request.Method, upstreamResponse); skipped {
		return r.skip(ctx, request.Address, upstreamResponse, reason), nil
	}
	if !contentencoding.IsDecodable(contentEncodingOf(upstreamResponse.Header)) {
		return r.refuse(ctx, request.Address, upstreamResponse, UndecodableEncoding), nil
	}
	return r.assessInReadingSlot(ctx, request, upstreamResponse)
}

func upstreamRequestFor(ctx context.Context, request proxiedrequest.Request) *http.Request {
	upstreamRequest := &http.Request{
		Method: request.Method,
		URL:    request.Address.WebAddress(),
		Header: request.Headers,
	}
	return upstreamRequest.WithContext(ctx)
}

func (r *RoundTripper) skip(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	upstreamResponse *http.Response,
	reason SkipReason,
) *http.Response {
	r.observers.AssessmentSkipped(ctx, address, reason)
	upstreamResponse.Header.Del(spamassessmenthttpheader.Name)
	return upstreamResponse
}

func contentEncodingOf(headers http.Header) string {
	return strings.ToLower(strings.TrimSpace(headers.Get("Content-Encoding")))
}

func (r *RoundTripper) refuse(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	upstreamResponse *http.Response,
	reason RefusalReason,
) *http.Response {
	_ = upstreamResponse.Body.Close()
	r.observers.PageRefused(ctx, address, reason)
	headers := http.Header{"Content-Length": {"0"}}
	if reason.isRetryable() {
		headers.Set("Retry-After", retryAfterSeconds)
	}
	return &http.Response{
		StatusCode: httpStatusesPerRefusal[reason],
		Header:     headers,
		Body:       http.NoBody,
	}
}

func (r *RoundTripper) assessInReadingSlot(
	ctx context.Context,
	request proxiedrequest.Request,
	upstreamResponse *http.Response,
) (*http.Response, error) {
	if !r.reserveReadingSlot(ctx, request.Address) {
		return r.endCancelledReading(ctx, request.Address, upstreamResponse, SlotWaitDeadline)
	}
	defer r.readingSlots.release()
	bodyPrefix, readFailure := r.bodyPrefixFrom(upstreamResponse.Body)
	if readFailure != nil {
		return r.endFailedReading(ctx, request.Address, upstreamResponse, readFailure)
	}
	page, refusal := r.assess(ctx, request, bodyPrefix, upstreamResponse.Header)
	if ctx.Err() != nil {
		return r.endCancelledReading(ctx, request.Address, upstreamResponse, refusal)
	}
	if refusal != "" {
		return r.refuse(ctx, request.Address, upstreamResponse, refusal), nil
	}
	return page.responseFrom(upstreamResponse), nil
}

func (r *RoundTripper) reserveReadingSlot(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
) bool {
	slotWaitStarted := r.clock.Now()
	reserved := r.readingSlots.reserve(ctx)
	r.observers.ReadingSlotWaited(ctx, address, r.clock.Now().Sub(slotWaitStarted))
	return reserved
}

func (r *RoundTripper) endCancelledReading(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	upstreamResponse *http.Response,
	deadlineReason RefusalReason,
) (*http.Response, error) {
	if errors.Is(context.Cause(ctx), roundtripcancel.ErrHeadersDeadlinePassed) {
		return r.refuse(ctx, address, upstreamResponse, deadlineReason), nil
	}
	_ = upstreamResponse.Body.Close()
	return nil, fmt.Errorf("reading the page ended: %w", context.Cause(ctx))
}

func (r *RoundTripper) bodyPrefixFrom(
	upstreamResponseBody io.Reader,
) ([]byte, error) {
	bodyPrefix, err := io.ReadAll(
		io.LimitReader(upstreamResponseBody, int64(r.pageByteCeiling)),
	)
	return bodyPrefix, err //nolint:wrapcheck // the round tripper reports the cause as the egress proxy gave it
}

func (r *RoundTripper) endFailedReading(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	upstreamResponse *http.Response,
	readFailure error,
) (*http.Response, error) {
	if ctx.Err() != nil {
		return r.endCancelledReading(ctx, address, upstreamResponse, BodyPrefixReadDeadline)
	}
	_ = upstreamResponse.Body.Close()
	r.observers.BodyPrefixReadFailed(ctx, address, readFailure)
	return &http.Response{
		StatusCode: http.StatusBadGateway,
		Header:     http.Header{"Content-Length": {"0"}},
		Body:       http.NoBody,
	}, nil
}

func (r *RoundTripper) assess(
	ctx context.Context,
	request proxiedrequest.Request,
	bodyPrefix []byte,
	upstreamResponseHeaders http.Header,
) (assessedPage, RefusalReason) {
	body, decoded := contentencoding.DecodedBodyFrom(
		bodyPrefix,
		contentEncodingOf(upstreamResponseHeaders),
		r.pageByteCeiling,
	)
	if !decoded {
		return assessedPage{}, UndecodableBody
	}
	assessment, outcome := r.assessmentRunner.Assess(
		ctx,
		request.Address,
		body,
		upstreamResponseHeaders,
	)
	if outcome != assessmentgate.Assessed {
		return assessedPage{}, refusalsPerOutcome[outcome]
	}
	r.observers.PageAssessed(ctx, request.Address, assessment)
	if ctx.Err() != nil || !r.clock.Now().Before(request.HeadersDeadline) {
		return assessedPage{}, AssessmentDeadline
	}
	return assessedPage{
		bodyPrefix:   bodyPrefix,
		wasBodyWhole: len(bodyPrefix) < r.pageByteCeiling,
		assessment:   assessment,
	}, ""
}
