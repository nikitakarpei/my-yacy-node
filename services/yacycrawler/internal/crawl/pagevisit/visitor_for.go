package pagevisit

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
)

type VisitorFor func(ignoredRefusals pagerefusals.IgnoredRefusals) Visitor

func New(
	fetches PageFetcher,
	recrawl RecrawlRule,
	htmlPageReading HTMLPageReading,
	refusalEnforcement RefusalEnforcementObserver,
	scrapeRequests ScrapeRequests,
) VisitorFor {
	return func(ignoredRefusals pagerefusals.IgnoredRefusals) Visitor {
		return &visitor{
			fetches:            fetches,
			recrawl:            recrawl,
			ignoredRefusals:    ignoredRefusals,
			htmlPageReading:    htmlPageReading,
			refusalEnforcement: refusalEnforcement,
			scrapeRequests:     scrapeRequests,
		}
	}
}
