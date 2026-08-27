package pagevisit

import "github.com/nikitakarpei/yacy-rwi-node/pagefetch"

type VisitorFor func(ignoredRefusals IgnoredRefusals) Visitor

func New(
	fetcher pagefetch.Fetcher,
	recrawl RecrawlRule,
	progress VisitProgress,
	scrapeRequests ScrapeRequests,
) VisitorFor {
	return func(ignoredRefusals IgnoredRefusals) Visitor {
		return &visitor{
			fetcher:         fetcher,
			recrawl:         recrawl,
			ignoredRefusals: ignoredRefusals,
			progress:        progress,
			scrapeRequests:  scrapeRequests,
		}
	}
}
