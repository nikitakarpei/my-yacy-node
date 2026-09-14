package pagevisit

type RecrawlRule interface {
	PageDueForRecrawl(lastVisit LastPageVisit) bool
}
