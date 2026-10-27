package classificationadding

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/classificationgate"
)

type RefusalReason string

const (
	BodyPrefixReadDeadline RefusalReason = "body_prefix_read_deadline"
	ClassificationDeadline RefusalReason = "classification_deadline"
	UndecodableEncoding    RefusalReason = "undecodable_encoding"
	UndecodableBody        RefusalReason = "undecodable_body"
	SlotWaitDeadline       RefusalReason = "slot_wait_deadline"
	UnclassifiedPage       RefusalReason = "unclassified_page"
)

var httpStatusesPerRefusal = map[RefusalReason]int{
	BodyPrefixReadDeadline: http.StatusGatewayTimeout,
	ClassificationDeadline: http.StatusGatewayTimeout,
	UndecodableEncoding:    http.StatusBadGateway,
	UndecodableBody:        http.StatusBadGateway,
	SlotWaitDeadline:       http.StatusServiceUnavailable,
	UnclassifiedPage:       http.StatusServiceUnavailable,
}

var refusalsPerOutcome = map[classificationgate.Outcome]RefusalReason{
	classificationgate.SlotWaitCancelled:       SlotWaitDeadline,
	classificationgate.ClassificationCancelled: ClassificationDeadline,
	classificationgate.Panicked:                UnclassifiedPage,
}

func (r RefusalReason) isRetryable() bool {
	return r == SlotWaitDeadline || r == UnclassifiedPage
}
