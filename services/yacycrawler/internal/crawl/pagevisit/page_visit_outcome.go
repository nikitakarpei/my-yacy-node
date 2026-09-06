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

type PageVisitOutcome struct {
	Conclusion     PageVisitConclusion
	DeferFor       time.Duration
	DiscoveredURLs []canonicalurl.CanonicalURL
	RedirectTarget canonicalurl.CanonicalURL
	Disposal       disposal.Reason
}

func disposedOutcome(reason disposal.Reason) PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitTerminal, Disposal: reason}
}

func crawledOutcome(discoveredURLs []canonicalurl.CanonicalURL) PageVisitOutcome {
	return PageVisitOutcome{
		Conclusion:     PageVisitTerminal,
		DiscoveredURLs: discoveredURLs,
		Disposal:       disposal.NotDisposed,
	}
}

func redirectedOutcome(redirectTarget canonicalurl.CanonicalURL) PageVisitOutcome {
	return PageVisitOutcome{
		Conclusion:     PageVisitTerminal,
		RedirectTarget: redirectTarget,
		Disposal:       disposal.NotDisposed,
	}
}

func deferredOutcome(deferFor time.Duration) PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitDeferred, DeferFor: deferFor}
}

func retryableOutcome() PageVisitOutcome {
	return PageVisitOutcome{Conclusion: PageVisitRetryable}
}
