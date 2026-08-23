package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type VisitorSource interface {
	VisitorFor(indexingRefusal IndexingRefusal) Visitor
}

type DocumentSource func(
	ctx context.Context,
	pageURL, contentType string,
	body []byte,
) (documentextraction.Document, error)

type ScrapeRequests interface {
	Publish(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) error
}

type visitorSource struct {
	fetcher        pagefetch.Fetcher
	recrawl        RecrawlRule
	documentSource DocumentSource
	observer       VisitProgress
	scrapeRequests ScrapeRequests
}

func New(
	fetcher pagefetch.Fetcher,
	recrawl RecrawlRule,
	documentSource DocumentSource,
	observer VisitProgress,
	scrapeRequests ScrapeRequests,
) VisitorSource {
	return &visitorSource{
		fetcher:        fetcher,
		recrawl:        recrawl,
		documentSource: documentSource,
		observer:       observer,
		scrapeRequests: scrapeRequests,
	}
}

func (s *visitorSource) VisitorFor(indexingRefusal IndexingRefusal) Visitor {
	return &visitor{
		fetcher:         s.fetcher,
		recrawl:         s.recrawl,
		documentSource:  s.documentSource,
		indexingRefusal: indexingRefusal,
		observer:        s.observer,
		scrapeRequests:  s.scrapeRequests,
	}
}
