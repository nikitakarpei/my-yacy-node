// Package retrydelay spaces repeated attempts apart, growing each delay up to a ceiling.
package retrydelay

import (
	"math"
	"time"
)

const growth = 1.5

type Bounds struct {
	Floor   time.Duration
	Ceiling time.Duration
}

func (b Bounds) Delay(attempt int) time.Duration {
	return time.Duration(math.Min(
		float64(b.Floor)*math.Pow(growth, float64(attempt-1)),
		float64(b.Ceiling),
	))
}
