package requestrelay

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
)

type RefusalReason string

const (
	WaitTooShort        RefusalReason = "wait_too_short"
	PageReadDeadline    RefusalReason = "page_read_deadline"
	AssessmentDeadline  RefusalReason = "assessment_deadline"
	UndecodableEncoding RefusalReason = "undecodable_encoding"
	UndecodableBody     RefusalReason = "undecodable_body"
	SlotWaitDeadline    RefusalReason = "slot_wait_deadline"
	UnassessedPage      RefusalReason = "unassessed_page"
)

var httpStatusesPerRefusal = map[RefusalReason]int{
	WaitTooShort:        http.StatusGatewayTimeout,
	PageReadDeadline:    http.StatusGatewayTimeout,
	AssessmentDeadline:  http.StatusGatewayTimeout,
	UndecodableEncoding: http.StatusBadGateway,
	UndecodableBody:     http.StatusBadGateway,
	SlotWaitDeadline:    http.StatusServiceUnavailable,
	UnassessedPage:      http.StatusServiceUnavailable,
}

var refusalsPerOutcome = map[assessmentgate.Outcome]RefusalReason{
	assessmentgate.SlotWaitDeadline:   SlotWaitDeadline,
	assessmentgate.AssessmentDeadline: AssessmentDeadline,
	assessmentgate.Panicked:           UnassessedPage,
}

func (r RefusalReason) isRetryable() bool {
	return r == SlotWaitDeadline || r == UnassessedPage
}
