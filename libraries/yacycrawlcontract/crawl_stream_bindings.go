package yacycrawlcontract

import "strings"

const (
	OrdersStreamName = "YACY_CRAWL_ORDERS"

	crawledPageStreamPrefix  = "YACY_CRAWL_PAGE_"
	crawledPageSubjectPrefix = "yacy.crawl.page."

	ReachedPagesStreamName = "CRAWL_REACHED_PAGES"
	ReachedPageSubject     = "crawl.reachedpage"
)

func CrawledPageStreamName(representation PageRepresentationKind) string {
	return crawledPageStreamPrefix + strings.ToUpper(string(representation))
}

func CrawledPageSubject(representation PageRepresentationKind) string {
	return crawledPageSubjectPrefix + string(representation)
}
