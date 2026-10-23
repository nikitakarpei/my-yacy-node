package yacycrawlcontract

type CrawlState string

const (
	CrawlStateRunning   CrawlState = "running"
	CrawlStateSucceeded CrawlState = "succeeded"
)
