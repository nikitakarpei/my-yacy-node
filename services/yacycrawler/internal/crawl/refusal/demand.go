// Package refusal names the demands a crawl target makes that the crawler honors.
package refusal

type Demand string

const (
	Cease Demand = "cease"
	Defer Demand = "defer"
)
