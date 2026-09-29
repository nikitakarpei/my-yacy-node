package requestrelay

import (
	"context"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
)

type pageAssessor struct {
	assessor        Assessor
	observers       Observers
	clock           Clock
	pageByteCeiling int
}

func (a pageAssessor) assess(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	bodyPrefix []byte,
	upstreamResponseHeaders http.Header,
	headersDueAt time.Time,
) (assessedPage, RefusalReason) {
	body, decoded := contentencoding.DecodedBodyFrom(
		bodyPrefix,
		contentEncodingOf(upstreamResponseHeaders),
		a.pageByteCeiling,
	)
	if !decoded {
		return assessedPage{}, UndecodableBody
	}
	assessment, outcome := a.assessor.Assess(
		ctx,
		address,
		body,
		upstreamResponseHeaders,
		headersDueAt,
	)
	if outcome != assessmentgate.Assessed {
		return assessedPage{}, refusalsPerOutcome[outcome]
	}
	a.observers.PageAssessed(ctx, address, assessment)
	if !a.clock.Now().Before(headersDueAt) {
		return assessedPage{}, AssessmentDeadline
	}
	return assessedPage{
		bodyPrefix:   bodyPrefix,
		wasBodyWhole: len(bodyPrefix) < a.pageByteCeiling,
		assessment:   assessment,
	}, ""
}
