package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

type PageVisitConclusion int

const (
	PageVisitTerminal PageVisitConclusion = iota
	PageVisitRetryable
	PageVisitDeferred
	PageVisitReturned
)

type PageVisitOutcome struct {
	Conclusion  PageVisitConclusion
	DeferFor    time.Duration
	Disposal    disposal.Reason
	ReturnCause error
}

func disposedOutcome(reason disposal.Reason) PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitTerminal, Disposal: reason}
}

func completedOutcome() PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitTerminal, Disposal: disposal.NotDisposed}
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
