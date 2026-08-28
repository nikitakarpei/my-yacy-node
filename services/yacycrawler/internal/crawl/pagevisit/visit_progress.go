package pagevisit

import "time"

type VisitProgress interface {
	FetchTook(duration time.Duration)
	PageFetched()
	AccessRefusalHonored()
	IndexingRefusalHonored()
	LinkDiscoveryRefusalHonored()
	ScrapeRequestPublished()
}
