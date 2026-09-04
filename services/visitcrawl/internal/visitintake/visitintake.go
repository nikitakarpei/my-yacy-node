// Package visitintake receives a visited-page visit on a signed link, places
// one crawl order for it, and redirects the browser onward without waiting for
// that placement's outcome. MountVisitIntake is its only surface.
package visitintake

import (
	"context"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const PathVisit = "/visit"

type CrawlOrderPlacer interface {
	Place(ctx context.Context, order yacycrawlcontract.CrawlOrder)
}

func MountVisitIntake(
	mux *http.ServeMux,
	placer CrawlOrderPlacer,
	profile yacycrawlcontract.CrawlProfile,
	linkSecret string,
) {
	mux.Handle(PathVisit, visitedPageEndpoint{
		placer:     placer,
		profile:    profile,
		linkSecret: linkSecret,
	})
}
