package pagevisit

type RecrawlRule interface {
	PageDueForRecrawl(lastVisit PageVisit) bool
}
