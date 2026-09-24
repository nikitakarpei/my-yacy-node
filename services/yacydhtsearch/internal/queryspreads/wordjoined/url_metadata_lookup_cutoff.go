package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type URLMetadataLookupCutoff struct {
	PercentOfDocuments int
	Grace              time.Duration
}

func (cutoff URLMetadataLookupCutoff) endedLookupFrom(
	outcomesAsTheySettle <-chan peerasks.URLMetadataAskOutcome,
	lookupInFlight urlMetadataLookupInFlight,
) endedURLMetadataLookup {
	var graceEnded <-chan time.Time
	for {
		select {
		case outcome, open := <-outcomesAsTheySettle:
			if !open {
				return lookupInFlight.endedBy(URLMetadataLookupEndedByEveryAskSettled)
			}
			lookupInFlight.settle(outcome)
			if lookupInFlight.covered() {
				return lookupInFlight.endedBy(URLMetadataLookupEndedByCoverage)
			}
			if graceEnded == nil && cutoff.reachedBy(lookupInFlight.settledShare()) {
				grace := time.NewTimer(cutoff.Grace)
				defer grace.Stop()
				graceEnded = grace.C
			}
		case <-graceEnded:
			return lookupInFlight.cutOff()
		}
	}
}

func (cutoff URLMetadataLookupCutoff) reachedBy(settledShare float64) bool {
	return cutoff.PercentOfDocuments > 0 &&
		settledShare >= float64(cutoff.PercentOfDocuments)/percentOfTheWhole
}

const percentOfTheWhole = 100
