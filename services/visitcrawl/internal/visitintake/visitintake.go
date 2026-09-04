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

type CrawlOrderPlacement interface {
	Start(order yacycrawlcontract.CrawlOrder)
}

func MountVisitIntake(
	mux *http.ServeMux,
	placement CrawlOrderPlacement,
	profile yacycrawlcontract.CrawlProfile,
	linkSecret string,
) {
	mux.Handle(PathVisit, visitedPageEndpoint{
		placement:  placement,
		profile:    profile,
		linkSecret: linkSecret,
	})
}
