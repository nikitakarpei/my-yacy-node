package pagevisit

import "time"

type Classification int

const (
	NotDue Classification = iota
	Succeeded
	Ceased
	Deferred
	NotAPage
	Transient
)

type VisitOutcome struct {
	Classification Classification
	DeferFor       time.Duration
	DiscoveredURLs []string
}
