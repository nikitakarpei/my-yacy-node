package pagevisit

import (
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type visitedPage struct {
	fetchOutcome pagefetch.FetchOutcome
	reading      pagehtmlreading.Reading
	readingErr   error
}

func (page visitedPage) isReadable() bool {
	return page.fetchOutcome.Status == pagefetch.FetchSucceeded && page.readingErr == nil
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
		return completedOutcome(pagevisitresult.Redirected)
	case pagefetch.FetchNotModified:
		return completedOutcome(pagevisitresult.NotModified)
	case pagefetch.FetchAccessRefused:
		return completedOutcome(pagevisitresult.AccessRefused)
	case pagefetch.FetchRejected, pagefetch.FetchGone:
		return completedOutcome(pagevisitresult.FetchRejected)
	case pagefetch.FetchRedirectTargetInvalid:
		return completedOutcome(pagevisitresult.RedirectTargetInvalid)
	case pagefetch.FetchOversized:
		return completedOutcome(pagevisitresult.Oversized)
	case pagefetch.FetchDeferred:
		return deferredOutcome(page.fetchOutcome.DeferFor)
	default:
		return retryableOutcome()
	}
}

func outcomeOfReading(readingErr error) PageVisitOutcome {
	if errors.Is(readingErr, pagehtmlreading.ErrPageNotHTML) {
		return completedOutcome(pagevisitresult.UnsupportedMediaType)
	}
	if readingErr != nil {
		return completedOutcome(pagevisitresult.UnreadableHTML)
	}
	return completedOutcome(pagevisitresult.Published)
}
