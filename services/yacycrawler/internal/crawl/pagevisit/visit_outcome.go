package pagevisit

import "time"

type VisitConclusion int

const (
	VisitCompleted VisitConclusion = iota
	VisitRetryable
	VisitDeferred
)

type VisitOutcome struct {
	Conclusion     VisitConclusion
	DeferFor       time.Duration
	DiscoveredURLs []string
}
