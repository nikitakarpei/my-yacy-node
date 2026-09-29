package requestrelay

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	spamassessmenthttpheader "github.com/nikitakarpei/yacy-rwi-node/spamassessment/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
)

type assessedPage struct {
	bodyPrefix   []byte
	wasBodyWhole bool
	assessment   spamassessment.Assessment
}

type pageAssessor struct {
	assessor        Assessor
	observers       Observers
	clock           Clock
	pageByteCeiling int
}

func (a pageAssessor) assessedPageFrom(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	bodyPrefix []byte,
	answerHeaders http.Header,
	headersDueAt time.Time,
) (assessedPage, RefusalReason) {
	body, decoded := contentencoding.DecodedBodyFrom(
		bodyPrefix,
		contentEncodingOf(answerHeaders),
		a.pageByteCeiling,
	)
	if !decoded {
		return assessedPage{}, UndecodableBody
	}
	assessment, outcome := a.assessmentFrom(ctx, address, body, answerHeaders, headersDueAt)
	if outcome != assessmentgate.Assessed {
		return assessedPage{}, refusalsPerOutcome[outcome]
	}
	if !a.clock.Now().Before(headersDueAt) {
		return assessedPage{}, AssessmentDeadline
	}
	return assessedPage{
		bodyPrefix:   bodyPrefix,
		wasBodyWhole: len(bodyPrefix) < a.pageByteCeiling,
		assessment:   assessment,
	}, ""
}

func (a pageAssessor) assessmentFrom(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	body []byte,
	answerHeaders http.Header,
	headersDueAt time.Time,
) (spamassessment.Assessment, assessmentgate.Outcome) {
	assessmentStarted := a.clock.Now()
	assessment, outcome := a.assessor.AssessmentFrom(
		ctx,
		address,
		body,
		answerHeaders,
		headersDueAt,
	)
	if outcome == assessmentgate.Assessed {
		a.observers.PageAssessed(ctx, address, assessment, a.clock.Now().Sub(assessmentStarted))
	}
	return assessment, outcome
}

func (p assessedPage) headersFrom(answerHeaders http.Header) http.Header {
	headers := relayedheaders.EndToEndHeadersOf(answerHeaders)
	if p.wasBodyWhole {
		headers.Set("Content-Length", strconv.Itoa(len(p.bodyPrefix)))
	}
	headers.Set(spamassessmenthttpheader.Name, spamassessmenthttpheader.ValueOf(p.assessment))
	return headers
}
