package pagevisit

import (
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/reachedpagepublication"
)

type VisitorSource interface {
	VisitorFor(indexingRefusal pageabsorption.IndexingRefusal) Visitor
}

type visitorSource struct {
	fetcher   pagefetch.Fetcher
	recrawl   RecrawlRule
	absorbers pageabsorption.AbsorberSource
	observer  VisitProgress
	reached   *reachedpagepublication.Publisher
}

func New(
	fetcher pagefetch.Fetcher,
	recrawl RecrawlRule,
	absorbers pageabsorption.AbsorberSource,
	observer VisitProgress,
	reached *reachedpagepublication.Publisher,
) VisitorSource {
	return &visitorSource{
		fetcher:   fetcher,
		recrawl:   recrawl,
		absorbers: absorbers,
		observer:  observer,
		reached:   reached,
	}
}

func (s *visitorSource) VisitorFor(
	indexingRefusal pageabsorption.IndexingRefusal,
) Visitor {
	return &visitor{
		fetcher:  s.fetcher,
		recrawl:  s.recrawl,
		absorber: s.absorbers.AbsorberFor(indexingRefusal),
		observer: s.observer,
		reached:  s.reached,
	}
}
