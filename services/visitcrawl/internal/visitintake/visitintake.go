// Package visitintake receives a visited-page visit on a signed link,
// attempts to place one crawl order for it, and redirects the browser onward
// without waiting for that attempt's outcome. MountVisitIntake is its only
// surface.
package visitintake

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const PathVisit = "/visit"

func MountVisitIntake(
	mux *http.ServeMux,
	startCrawlOrderPlacementAttempt func(order yacycrawlcontract.CrawlOrder),
	profile yacycrawlcontract.CrawlProfile,
	visitedPageObserver VisitedPageObserver,
	linkSecret string,
) {
	mux.Handle(PathVisit, visitedPageEndpoint{
		startCrawlOrderPlacementAttempt: startCrawlOrderPlacementAttempt,
		profile:                         profile,
		observer:                        visitedPageObserver,
		linkSecret:                      linkSecret,
	})
}
