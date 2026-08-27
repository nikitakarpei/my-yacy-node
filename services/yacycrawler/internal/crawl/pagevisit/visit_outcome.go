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

type VisitOutcome struct {
	Conclusion     VisitConclusion
	DeferFor       time.Duration
	DiscoveredURLs []canonicalurl.CanonicalURL
	Disposal       disposal.Reason
}

func completedVisit(
	reason disposal.Reason,
	discoveredURLs []canonicalurl.CanonicalURL,
) VisitOutcome {
	return VisitOutcome{
		Conclusion:     VisitCompleted,
		DiscoveredURLs: discoveredURLs,
		Disposal:       reason,
	}
}

func deferredVisit(deferFor time.Duration) VisitOutcome {
	return VisitOutcome{Conclusion: VisitDeferred, DeferFor: deferFor}
}

func retryableVisit() VisitOutcome {
	return VisitOutcome{Conclusion: VisitRetryable}
}
