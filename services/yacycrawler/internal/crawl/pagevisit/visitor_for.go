package pagevisit

import (
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
)

type VisitorFor func(ignoredRefusals pagerefusals.IgnoredRefusals) Visitor

func New(
	fetcher pagefetch.Fetcher,
	recrawl RecrawlRule,
	progress VisitProgress,
	scrapeRequests ScrapeRequests,
) VisitorFor {
	return func(ignoredRefusals pagerefusals.IgnoredRefusals) Visitor {
		return &visitor{
			fetcher:         fetcher,
			recrawl:         recrawl,
			ignoredRefusals: ignoredRefusals,
			progress:        progress,
			scrapeRequests:  scrapeRequests,
		}
	}
}
