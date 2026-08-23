package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagelinks"
)

type VisitorSource interface {
	VisitorFor(indexingRefusal IndexingRefusal) Visitor
}

type PageLinksSource func(
	ctx context.Context,
	pageURL, contentType string,
	body []byte,
) (pagelinks.PageLinks, error)

type ScrapeRequests interface {
	Publish(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) error
}

type visitorSource struct {
	fetcher        pagefetch.Fetcher
	recrawl        RecrawlRule
	pageLinks      PageLinksSource
	observer       VisitProgress
	scrapeRequests ScrapeRequests
}

func New(
	fetcher pagefetch.Fetcher,
	recrawl RecrawlRule,
	pageLinks PageLinksSource,
	observer VisitProgress,
	scrapeRequests ScrapeRequests,
) VisitorSource {
	return &visitorSource{
		fetcher:        fetcher,
		recrawl:        recrawl,
		pageLinks:      pageLinks,
		observer:       observer,
		scrapeRequests: scrapeRequests,
	}
}

func (s *visitorSource) VisitorFor(indexingRefusal IndexingRefusal) Visitor {
	return &visitor{
		fetcher:         s.fetcher,
		recrawl:         s.recrawl,
		pageLinks:       s.pageLinks,
		indexingRefusal: indexingRefusal,
		observer:        s.observer,
		scrapeRequests:  s.scrapeRequests,
	}
}
