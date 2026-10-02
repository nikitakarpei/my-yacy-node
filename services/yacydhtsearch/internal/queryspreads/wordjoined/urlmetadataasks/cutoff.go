package urlmetadataasks

import "time"

type Cutoff struct {
	PercentOfDocuments int
	Grace              time.Duration
}

func (cutoff Cutoff) reachedBy(settledShare float64) bool {
	return cutoff.PercentOfDocuments > 0 &&
		settledShare >= float64(cutoff.PercentOfDocuments)/percentOfTheWhole
}

const percentOfTheWhole = 100
