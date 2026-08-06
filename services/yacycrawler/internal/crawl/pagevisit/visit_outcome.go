package pagevisit

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

type VisitConclusion int

const (
	VisitCompleted VisitConclusion = iota
	VisitRetryable
	VisitDeferred
)

type VisitOutcome struct {
	Conclusion     VisitConclusion
	DeferFor       time.Duration
	Fetched        bool
	DiscoveredURLs []string
	Disposal       disposal.Reason
}
