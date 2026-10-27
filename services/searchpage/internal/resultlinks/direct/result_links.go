// Package direct links each result straight to its page, for a search page without visitcrawl.
package direct

type ResultLinks struct{}

func New() ResultLinks {
	return ResultLinks{}
}

func (ResultLinks) LinkFor(page string) string {
	return page
}
