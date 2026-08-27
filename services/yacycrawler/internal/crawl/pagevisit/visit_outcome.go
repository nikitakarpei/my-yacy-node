package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

type VisitConclusion int

const (
	VisitCompleted VisitConclusion = iota
	VisitRetryable
	VisitDeferred
)

var noDiscoveredURLs []canonicalurl.CanonicalURL

type VisitOutcome struct {
	Conclusion     VisitConclusion
	DeferFor       time.Duration
	DiscoveredURLs []canonicalurl.CanonicalURL
	Disposal       disposal.Reason
}

func (outcome VisitOutcome) Disposed() bool {
	return outcome.Disposal.Disposes()
}

func completedOutcome(
	reason disposal.Reason,
	discoveredURLs []canonicalurl.CanonicalURL,
) VisitOutcome {
	return VisitOutcome{
		Conclusion:     VisitCompleted,
		DiscoveredURLs: discoveredURLs,
		Disposal:       reason,
	}
}

func deferredOutcome(deferFor time.Duration) VisitOutcome {
	return VisitOutcome{Conclusion: VisitDeferred, DeferFor: deferFor}
}

func retryableOutcome() VisitOutcome {
	return VisitOutcome{Conclusion: VisitRetryable}
}
