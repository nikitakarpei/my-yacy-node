package pagerelay

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readtimer"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/replydeadlines"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
)

const retryAfterSeconds = "1"

var htmlMediaTypes = map[string]bool{"text/html": true, "application/xhtml+xml": true}

type exchange struct {
	egress          Egress
	assessor        Assessor
	observers       Observers
	limits          Limits
	clock           Clock
	assessmentSlots assessmentSlots
	address         canonicalurl.CanonicalURL
	timer           *readtimer.Timer
	reply           Reply
}

func (e exchange) relayFromEgress(
	ctx context.Context,
	egressRequest *http.Request,
	deadlines replydeadlines.Deadlines,
) {
	e.timer.EndReadingAt(deadlines.ReadingEndsAt)
	answer, err := e.egress.RoundTrip(egressRequest)
	if err != nil {
		e.failReading(ctx, err)
		return
	}
	defer func() { _ = answer.Body.Close() }()
	e.relayAnswer(ctx, egressRequest.Method, answer, deadlines)
}

func (e exchange) refuse(ctx context.Context, reason RefusalReason) {
	e.observers.AnswerRefused(ctx, e.address, reason)
	headers := http.Header{"Content-Length": {"0"}}
	if reason.isRetryable() {
		headers.Set("Retry-After", retryAfterSeconds)
	}
	e.reply.SendHead(refusalStatuses[reason], headers)
}

func (e exchange) failReading(ctx context.Context, cause error) {
	if e.timer.Expired() {
		e.refuse(ctx, PageReadTimeout)
		return
	}
	e.observers.RelayFailed(ctx, e.address, cause)
	e.reply.SendHead(http.StatusBadGateway, http.Header{"Content-Length": {"0"}})
}

func (e exchange) relayAnswer(
	ctx context.Context,
	method string,
	answer *http.Response,
	deadlines replydeadlines.Deadlines,
) {
	if e.clock.Now().After(deadlines.ReadingEndsAt) {
		e.refuse(ctx, PageReadTimeout)
		return
	}
	if reason, isNonPage := nonPageReasonOf(method, answer); isNonPage {
		e.relayNonPage(ctx, reason, answer)
		return
	}
	if !contentencoding.IsDecodable(contentEncodingOf(answer.Header)) {
		e.refuse(ctx, UndecodableEncoding)
		return
	}
	e.relayPage(ctx, answer, deadlines)
}

func nonPageReasonOf(method string, answer *http.Response) (NonPageReason, bool) {
	if method == http.MethodHead {
		return HeadRequest, true
	}
	if answer.StatusCode < http.StatusOK || answer.StatusCode >= http.StatusMultipleChoices {
		return Not2xx, true
	}
	if mediaType, _, _ := mime.ParseMediaType(
		answer.Header.Get("Content-Type"),
	); !htmlMediaTypes[mediaType] {
		return NotHTML, true
	}
	return "", false
}

func contentEncodingOf(headers http.Header) string {
	return strings.ToLower(strings.TrimSpace(headers.Get("Content-Encoding")))
}

func (e exchange) relayNonPage(ctx context.Context, reason NonPageReason, answer *http.Response) {
	e.observers.NonPageRelayed(ctx, e.address, reason)
	e.reply.SendHead(answer.StatusCode, relayedheaders.EndToEndHeadersOf(answer.Header))
	e.relayRest(ctx, answer.Body, nil)
}

func (e exchange) relayPage(
	ctx context.Context,
	answer *http.Response,
	deadlines replydeadlines.Deadlines,
) {
	if !e.assessmentSlots.tryReserve() {
		e.refuse(ctx, TooManyAtOnce)
		return
	}
	bodyPrefix, readFailure := e.bodyPrefixFrom(answer.Body)
	if readFailure != nil {
		e.assessmentSlots.release()
		e.failReading(ctx, readFailure)
		return
	}
	page, refusal := e.assessedPageFrom(ctx, bodyPrefix, answer.Header, deadlines)
	e.assessmentSlots.release()
	if refusal != "" {
		e.refuse(ctx, refusal)
		return
	}
	e.reply.SendHead(answer.StatusCode, page.headersFrom(answer.Header))
	e.relayRest(ctx, answer.Body, page.bodyPrefix)
}

func (e exchange) bodyPrefixFrom(answerBody io.Reader) ([]byte, error) {
	bodyPrefix, err := io.ReadAll(io.LimitReader(answerBody, int64(e.limits.PageByteCeiling)))
	e.timer.Stop()
	return bodyPrefix, err //nolint:wrapcheck // the relay reports the cause as the egress gave it
}

func (e exchange) relayRest(ctx context.Context, answerBody io.Reader, bodyPrefix []byte) {
	rest := e.timer.IdleLimitedFrom(answerBody, e.limits.RelayIdleTimeout)
	if _, err := io.Copy(e.reply, io.MultiReader(bytes.NewReader(bodyPrefix), rest)); err != nil {
		e.observers.RelayFailed(ctx, e.address, err)
		e.reply.CutShort()
	}
}

type assessedPage struct {
	bodyPrefix   []byte
	wasBodyWhole bool
	assessment   spamassessment.Assessment
}

func (e exchange) assessedPageFrom(
	ctx context.Context,
	bodyPrefix []byte,
	answerHeaders http.Header,
	deadlines replydeadlines.Deadlines,
) (assessedPage, RefusalReason) {
	body, decoded := contentencoding.DecodedBodyFrom(
		bodyPrefix,
		contentEncodingOf(answerHeaders),
		e.limits.PageByteCeiling,
	)
	if !decoded {
		return assessedPage{}, UndecodableBody
	}
	assessment, assessed := e.assessmentFrom(ctx, body, answerHeaders, deadlines.HeadersDueAt)
	if !e.clock.Now().Before(deadlines.HeadersDueAt) {
		return assessedPage{}, ResponseHeaderTimeout
	}
	if !assessed {
		return assessedPage{}, UnassessedPage
	}
	return assessedPage{
		bodyPrefix:   bodyPrefix,
		wasBodyWhole: len(bodyPrefix) < e.limits.PageByteCeiling,
		assessment:   assessment,
	}, ""
}

func (e exchange) assessmentFrom(
	ctx context.Context,
	body []byte,
	answerHeaders http.Header,
	headersDueAt time.Time,
) (spamassessment.Assessment, bool) {
	assessmentStarted := e.clock.Now()
	assessment, assessed := e.assessor.AssessmentFrom(
		ctx,
		e.address,
		body,
		answerHeaders,
		headersDueAt,
	)
	if assessed {
		e.observers.PageAssessed(ctx, e.address, assessment, e.clock.Now().Sub(assessmentStarted))
	}
	return assessment, assessed
}

func (p assessedPage) headersFrom(answerHeaders http.Header) http.Header {
	headers := relayedheaders.EndToEndHeadersOf(answerHeaders)
	if p.wasBodyWhole {
		headers.Set("Content-Length", strconv.Itoa(len(p.bodyPrefix)))
	}
	headers.Set(spamassessment.HeaderName, p.assessment.HeaderValue())
	return headers
}
