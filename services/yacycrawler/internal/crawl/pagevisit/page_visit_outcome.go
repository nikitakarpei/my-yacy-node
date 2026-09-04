package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

type PageVisitConclusion int

const (
	PageVisitTerminal PageVisitConclusion = iota
	PageVisitRetryable
	PageVisitDeferred
)

var noDiscoveredURLs []canonicalurl.CanonicalURL

type PageVisitOutcome struct {
	Conclusion     PageVisitConclusion
	DeferFor       time.Duration
	DiscoveredURLs []canonicalurl.CanonicalURL
	Disposal       disposal.Reason
}

func terminalOutcome(
	reason disposal.Reason,
	discoveredURLs []canonicalurl.CanonicalURL,
) PageVisitOutcome {
	return PageVisitOutcome{
		Conclusion:     PageVisitTerminal,
		DiscoveredURLs: discoveredURLs,
		Disposal:       reason,
	}
}

func deferredOutcome(deferFor time.Duration) PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitDeferred, DeferFor: deferFor}
}

func retryableOutcome() PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitRetryable}
}
