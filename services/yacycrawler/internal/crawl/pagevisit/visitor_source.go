package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type VisitorSource interface {
	VisitorFor(indexingRefusal IndexingRefusal) Visitor
}

type ScrapeRequests interface {
	Publish(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) error
}

type visitorSource struct {
	fetcher        pagefetch.Fetcher
	recrawl        RecrawlRule
	observer       VisitProgress
	scrapeRequests ScrapeRequests
}

func New(
	fetcher pagefetch.Fetcher,
	recrawl RecrawlRule,
	observer VisitProgress,
	scrapeRequests ScrapeRequests,
) VisitorSource {
	return &visitorSource{
		fetcher:        fetcher,
		recrawl:        recrawl,
		observer:       observer,
		scrapeRequests: scrapeRequests,
	}
}

func (s *visitorSource) VisitorFor(indexingRefusal IndexingRefusal) Visitor {
	return &visitor{
		fetcher:         s.fetcher,
		recrawl:         s.recrawl,
		indexingRefusal: indexingRefusal,
		observer:        s.observer,
		scrapeRequests:  s.scrapeRequests,
	}
}
