package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/scraperequestpublication"
)

type VisitorSource interface {
	VisitorFor(indexingRefusal IndexingRefusal) Visitor
}

type PageExtractor interface {
	DocumentFrom(
		ctx context.Context,
		pageURL, contentType string,
		body []byte,
	) (documentextraction.Document, error)
}

type visitorSource struct {
	fetcher        pagefetch.Fetcher
	recrawl        RecrawlRule
	extractor      PageExtractor
	observer       VisitProgress
	scrapeRequests *scraperequestpublication.Publisher
}

func New(
	fetcher pagefetch.Fetcher,
	recrawl RecrawlRule,
	extractor PageExtractor,
	observer VisitProgress,
	scrapeRequests *scraperequestpublication.Publisher,
) VisitorSource {
	return &visitorSource{
		fetcher:        fetcher,
		recrawl:        recrawl,
		extractor:      extractor,
		observer:       observer,
		scrapeRequests: scrapeRequests,
	}
}

func (s *visitorSource) VisitorFor(indexingRefusal IndexingRefusal) Visitor {
	return &visitor{
		fetcher:         s.fetcher,
		recrawl:         s.recrawl,
		extractor:       s.extractor,
		indexingRefusal: indexingRefusal,
		observer:        s.observer,
		scrapeRequests:  s.scrapeRequests,
	}
}
