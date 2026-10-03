package wordjoined

import (
	"time"
)

type URLMetadataLookupCutoff struct {
	PercentOfDocuments int
	Grace              time.Duration
}

func (cutoff URLMetadataLookupCutoff) reachedBy(settledShare float64) bool {
	return cutoff.PercentOfDocuments > 0 &&
		settledShare >= float64(cutoff.PercentOfDocuments)/percentOfTheWhole
}

const percentOfTheWhole = 100
