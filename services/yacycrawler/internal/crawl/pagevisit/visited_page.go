package pagevisit

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type visitedPage struct {
	url          canonicalurl.CanonicalURL
	fetchOutcome pagefetch.FetchOutcome
	reading      pagereading.Reading
	readingErr   error
}

func (page visitedPage) isReadable() bool {
	return page.fetchOutcome.Status == pagefetch.FetchSucceeded && page.readingErr == nil
}

func (page visitedPage) refusesIndexing() bool {
	return page.reading.Refusals.RefusesIndexing
}

func (page visitedPage) crawledPage() yacycrawlcontract.CrawledPage {
	return yacycrawlcontract.CrawledPage{
		PageURL:     page.url,
		ContentType: page.fetchOutcome.Page.ContentType,
	}
}

func (page visitedPage) nextPageVisitsFrom(
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) []pendingpagevisit.PendingPageVisit {
	switch page.fetchOutcome.Status {
	case pagefetch.FetchSucceeded:
		linkedPageVisits := make(
			[]pendingpagevisit.PendingPageVisit, 0, len(page.reading.DiscoveredURLs),
		)
		for _, url := range page.reading.DiscoveredURLs {
			linkedPageVisits = append(linkedPageVisits, pendingPageVisit.LinkedPageVisitTo(url))
		}
		return linkedPageVisits
	case pagefetch.FetchRedirected:
		return []pendingpagevisit.PendingPageVisit{
			pendingPageVisit.RedirectedPageVisitTo(page.fetchOutcome.RedirectTarget),
		}
	default:
		return nil
	}
}

func (page visitedPage) startsRecrawlGrace() bool {
	switch page.fetchOutcome.Status {
	case pagefetch.FetchSucceeded,
		pagefetch.FetchNotModified,
		pagefetch.FetchRedirected,
		pagefetch.FetchAccessRefused:
		return true
	default:
		return false
	}
}

func (page visitedPage) outcome() PageVisitOutcome {
	switch page.fetchOutcome.Status {
	case pagefetch.FetchSucceeded:
		return outcomeOfReading(page.readingErr)
	case pagefetch.FetchRedirected:
		return completedOutcome(yacycrawlcontract.ResultRedirected)
	case pagefetch.FetchNotModified:
		return completedOutcome(yacycrawlcontract.ResultNotModified)
	case pagefetch.FetchAccessRefused:
		return completedOutcome(yacycrawlcontract.ResultAccessRefused)
	case pagefetch.FetchRejected:
		return completedOutcome(yacycrawlcontract.ResultFetchRejected)
	case pagefetch.FetchGone:
		return completedOutcome(yacycrawlcontract.ResultNotFound)
	case pagefetch.FetchRedirectTargetInvalid:
		return completedOutcome(yacycrawlcontract.ResultRedirectTargetInvalid)
	case pagefetch.FetchOversized:
		return completedOutcome(yacycrawlcontract.ResultOversized)
	case pagefetch.FetchDeferred:
		return deferredOutcome(page.fetchOutcome.DeferFor)
	default:
		return retryableOutcome()
	}
}

func outcomeOfReading(readingErr error) PageVisitOutcome {
	if readingErr != nil {
		return completedOutcome(yacycrawlcontract.ResultUnreadableHTML)
	}
	return completedOutcome(yacycrawlcontract.ResultPublished)
}
