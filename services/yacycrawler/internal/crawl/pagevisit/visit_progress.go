package pagevisit

type VisitProgress interface {
	PageFetched()
	AccessRefusalHonored()
	IndexingRefusalHonored()
	LinkDiscoveryRefusalHonored()
	ScrapeRequestPublished()
}
