// Package budgetbuckets gives the histogram buckets of a duration that a
// budget cuts off: fine steps up to the budget, one just past it for work that
// stops at the deadline, and a few for work that overruns it.
package budgetbuckets

import (
	"math"
	"time"
)

const (
	bucketRatio                  = 1.25
	amountOfBucketsUpToTheBudget = 32
)

var overBudgetShares = []float64{1.01, 1.25, 1.5, 2}

func DurationBucketsFor(budget time.Duration) []float64 {
	seconds := budget.Seconds()
	buckets := make([]float64, 0, amountOfBucketsUpToTheBudget+len(overBudgetShares))
	for step := amountOfBucketsUpToTheBudget - 1; step >= 0; step-- {
		buckets = append(buckets, seconds/math.Pow(bucketRatio, float64(step)))
	}
	for _, share := range overBudgetShares {
		buckets = append(buckets, seconds*share)
	}

	return buckets
}
