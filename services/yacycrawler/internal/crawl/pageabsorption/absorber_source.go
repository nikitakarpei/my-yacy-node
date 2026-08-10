package pageabsorption

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
)

type AbsorberSource interface {
	AbsorberFor(indexingRefusal IndexingRefusal) Absorber
}

type absorberSource struct {
	extractor PageExtractor
	publisher PagePublisher
	clock     clock.Clock
}

func New(
	extractor PageExtractor,
	publisher PagePublisher,
	clock clock.Clock,
) AbsorberSource {
	return &absorberSource{
		extractor: extractor,
		publisher: publisher,
		clock:     clock,
	}
}

func (s *absorberSource) AbsorberFor(indexingRefusal IndexingRefusal) Absorber {
	return &absorber{
		extractor:       s.extractor,
		publisher:       s.publisher,
		clock:           s.clock,
		indexingRefusal: indexingRefusal,
	}
}
