package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/reachedpagepublication"
)

type VisitorSource interface {
	VisitorFor(indexingRefusal IndexingRefusal) Visitor
}

type PageExtractor interface {
	DocumentFrom(
		ctx context.Context,
		pageURL, contentType string,
		body []byte,
	) (contentextraction.ExtractedDocument, error)
}

type visitorSource struct {
	fetcher   pagefetch.Fetcher
	recrawl   RecrawlRule
	extractor PageExtractor
	observer  VisitProgress
	reached   *reachedpagepublication.Publisher
}

func New(
	fetcher pagefetch.Fetcher,
	recrawl RecrawlRule,
	extractor PageExtractor,
	observer VisitProgress,
	reached *reachedpagepublication.Publisher,
) VisitorSource {
	return &visitorSource{
		fetcher:   fetcher,
		recrawl:   recrawl,
		extractor: extractor,
		observer:  observer,
		reached:   reached,
	}
}

func (s *visitorSource) VisitorFor(indexingRefusal IndexingRefusal) Visitor {
	return &visitor{
		fetcher:         s.fetcher,
		recrawl:         s.recrawl,
		extractor:       s.extractor,
		indexingRefusal: indexingRefusal,
		observer:        s.observer,
		reached:         s.reached,
	}
}
