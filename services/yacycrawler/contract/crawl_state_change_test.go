package yacycrawlcontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestEachCrawlStateChangeChangesLeavesOrRefusesEveryState(t *testing.T) {
	const (
		changed   = "changed"
		unchanged = "unchanged"
		refused   = "refused"
	)
	outcomeOf := func(change yacycrawlcontract.CrawlStateChange, state yacycrawlcontract.CrawlState) string {
		switch {
		case change.Changes(state):
			return changed
		case change.Refuses(state):
			return refused
		default:
			return unchanged
		}
	}
	wantOutcomes := map[string]struct {
		change         yacycrawlcontract.CrawlStateChange
		resultingState yacycrawlcontract.CrawlState
		outcomes       []string
	}{
		"pause": {
			yacycrawlcontract.CrawlPause, yacycrawlcontract.CrawlStatePaused,
			[]string{changed, unchanged, refused, refused, refused},
		},
		"resumption": {
			yacycrawlcontract.CrawlResumption, yacycrawlcontract.CrawlStateRunning,
			[]string{unchanged, changed, refused, refused, refused},
		},
		"cancellation": {
			yacycrawlcontract.CrawlCancellation, yacycrawlcontract.CrawlStateCancelling,
			[]string{changed, changed, unchanged, refused, unchanged},
		},
	}
	for name, want := range wantOutcomes {
		if want.change.ResultingState() != want.resultingState {
			t.Errorf(
				"%s leads to %s, want %s",
				name,
				want.change.ResultingState(),
				want.resultingState,
			)
		}
		for position, state := range yacycrawlcontract.CrawlStates {
			if outcome := outcomeOf(want.change, state); outcome != want.outcomes[position] {
				t.Errorf(
					"%s of a %s crawl is %s, want %s",
					name,
					state,
					outcome,
					want.outcomes[position],
				)
			}
		}
	}
}
