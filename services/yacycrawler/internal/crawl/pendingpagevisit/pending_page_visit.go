// Package pendingpagevisit names one URL a crawl still owes a page visit, as the
// frontier holds it from the worker that found it to the worker that pays it. A page
// links to visits one level deeper, and redirects to a visit at its own depth. A worker
// holds the visit it takes leased until a time.
package pendingpagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

type PendingPageVisit struct {
	CrawlID crawlid.CrawlID
	URL     canonicalurl.CanonicalURL
	Depth   int
}

type LeasedPageVisit struct {
	PageVisit   PendingPageVisit
	LeasedUntil time.Time
}

func SeedPageVisitsOf(
	crawlID crawlid.CrawlID,
	seedURLs []canonicalurl.CanonicalURL,
) []PendingPageVisit {
	seedPageVisits := make([]PendingPageVisit, 0, len(seedURLs))
	for _, seed := range seedURLs {
		seedPageVisits = append(seedPageVisits, PendingPageVisit{CrawlID: crawlID, URL: seed})
	}
	return seedPageVisits
}

func (pageVisit PendingPageVisit) LinkedPageVisitTo(
	url canonicalurl.CanonicalURL,
) PendingPageVisit {
	return PendingPageVisit{CrawlID: pageVisit.CrawlID, URL: url, Depth: pageVisit.Depth + 1}
}

func (pageVisit PendingPageVisit) RedirectedPageVisitTo(
	redirectTarget canonicalurl.CanonicalURL,
) PendingPageVisit {
	return PendingPageVisit{CrawlID: pageVisit.CrawlID, URL: redirectTarget, Depth: pageVisit.Depth}
}
