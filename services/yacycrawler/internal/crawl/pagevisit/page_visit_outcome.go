package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type PageVisitConclusion int

const (
	PageVisitCompleted PageVisitConclusion = iota
	PageVisitDropped
	PageVisitRetryable
	PageVisitDeferred
	PageVisitReturned
)

type PageVisitOutcome struct {
	Conclusion  PageVisitConclusion
	Result      yacycrawlcontract.PageVisitResult
	DropReason  yacycrawlcontract.PageVisitDropReason
	DeferFor    time.Duration
	ReturnCause error
}

func completedOutcome(result yacycrawlcontract.PageVisitResult) PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitCompleted, Result: result}
}

func droppedOutcome(reason yacycrawlcontract.PageVisitDropReason) PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitDropped, DropReason: reason}
}

func deferredOutcome(deferFor time.Duration) PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitDeferred, DeferFor: deferFor}
}

func retryableOutcome() PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitRetryable}
}

func returnedOutcome(cause error) PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitReturned, ReturnCause: cause}
}
