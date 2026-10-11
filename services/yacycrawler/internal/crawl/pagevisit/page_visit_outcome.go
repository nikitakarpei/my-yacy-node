package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitresult"
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
	Result      pagevisitresult.Result
	DropReason  pagevisitdrop.Reason
	DeferFor    time.Duration
	ReturnCause error
}

func completedOutcome(result pagevisitresult.Result) PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitCompleted, Result: result}
}

func droppedOutcome(reason pagevisitdrop.Reason) PageVisitOutcome {
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
