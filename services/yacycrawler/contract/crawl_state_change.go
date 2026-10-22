package yacycrawlcontract

import "slices"

type CrawlStateChange struct {
	from        []CrawlState
	to          CrawlState
	unchangedAt []CrawlState
}

var (
	CrawlPause = CrawlStateChange{
		from:        []CrawlState{CrawlStateRunning},
		to:          CrawlStatePaused,
		unchangedAt: []CrawlState{CrawlStatePaused},
	}
	CrawlResumption = CrawlStateChange{
		from:        []CrawlState{CrawlStatePaused},
		to:          CrawlStateRunning,
		unchangedAt: []CrawlState{CrawlStateRunning},
	}
	CrawlCancellation = CrawlStateChange{
		from:        []CrawlState{CrawlStateRunning, CrawlStatePaused},
		to:          CrawlStateCancelling,
		unchangedAt: []CrawlState{CrawlStateCancelling, CrawlStateCancelled},
	}
)

func (change CrawlStateChange) Changes(state CrawlState) bool {
	return slices.Contains(change.from, state)
}

func (change CrawlStateChange) ResultingState() CrawlState {
	return change.to
}

func (change CrawlStateChange) Refuses(state CrawlState) bool {
	return !change.Changes(state) && !slices.Contains(change.unchangedAt, state)
}
