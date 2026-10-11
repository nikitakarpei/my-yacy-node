package pagevisit

import (
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
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
		return completedOutcome()
	case pagefetch.FetchNotModified:
		return disposedOutcome(disposal.NotModified)
	case pagefetch.FetchAccessRefused:
		return disposedOutcome(disposal.AccessRefused)
	case pagefetch.FetchRejected, pagefetch.FetchGone:
		return disposedOutcome(disposal.FetchRejected)
	case pagefetch.FetchRedirectTargetInvalid:
		return disposedOutcome(disposal.RedirectTargetInvalid)
	case pagefetch.FetchOversized:
		return disposedOutcome(disposal.Oversized)
	case pagefetch.FetchDeferred:
		return deferredOutcome(page.fetchOutcome.DeferFor)
	default:
		return retryableOutcome()
	}
}

func outcomeOfReading(readingErr error) PageVisitOutcome {
	if errors.Is(readingErr, pagehtmlreading.ErrPageNotHTML) {
		return disposedOutcome(disposal.UnsupportedMediaType)
	}
	if readingErr != nil {
		return disposedOutcome(disposal.UnreadableHTML)
	}
	return completedOutcome()
}
