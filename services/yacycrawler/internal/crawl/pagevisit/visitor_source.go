package pagevisit

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
)

type VisitorSource interface {
	VisitorFor(indexingRefusal pageabsorption.IndexingRefusal) Visitor
}

type visitorSource struct {
	fetcher   Fetcher
	recrawl   RecrawlRule
	absorbers pageabsorption.AbsorberSource
	observer  VisitProgress
	clock     clock.Clock
}

func New(
	fetcher Fetcher,
	recrawl RecrawlRule,
	absorbers pageabsorption.AbsorberSource,
	observer VisitProgress,
	clock clock.Clock,
) VisitorSource {
	return &visitorSource{
		fetcher:   fetcher,
		recrawl:   recrawl,
		absorbers: absorbers,
		observer:  observer,
		clock:     clock,
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
		clock:    s.clock,
	}
}
