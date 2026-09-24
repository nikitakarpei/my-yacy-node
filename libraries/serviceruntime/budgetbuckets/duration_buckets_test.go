package budgetbuckets_test

import (
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/budgetbuckets"
)

const budget = 3 * time.Second

func TestTheBudgetBoundsABucket(t *testing.T) {
	t.Parallel()

	buckets := budgetbuckets.DurationBucketsFor(budget)

	if !slices.Contains(buckets, budget.Seconds()) {
		t.Fatalf("no bucket ends at the budget: %v", buckets)
	}
}

func TestADurationStoppedAtTheDeadlineStaysNearTheBudget(t *testing.T) {
	t.Parallel()

	buckets := budgetbuckets.DurationBucketsFor(budget)
	stoppedAtTheDeadline := (budget + time.Millisecond).Seconds()

	bucketIndex, _ := slices.BinarySearch(buckets, stoppedAtTheDeadline)
	if buckets[bucketIndex] > budget.Seconds()*1.01 {
		t.Fatalf(
			"a duration of %v falls in the bucket up to %v",
			stoppedAtTheDeadline,
			buckets[bucketIndex],
		)
	}
}

func TestUpToTheBudgetEachBucketIsAtMostAQuarterWiderThanTheOneBelow(t *testing.T) {
	t.Parallel()

	buckets := budgetbuckets.DurationBucketsFor(budget)
	budgetIndex := slices.Index(buckets, budget.Seconds())

	for index := 1; index <= budgetIndex; index++ {
		ratio := buckets[index] / buckets[index-1]
		if ratio <= 1 || ratio > 1.25+1e-9 {
			t.Fatalf("bucket %v follows %v", buckets[index], buckets[index-1])
		}
	}
}

func TestTheBucketsSpanAThousandthOfTheBudgetToTwiceIt(t *testing.T) {
	t.Parallel()

	buckets := budgetbuckets.DurationBucketsFor(budget)

	if buckets[0] > budget.Seconds()/1000 {
		t.Fatalf("the smallest bucket ends at %v", buckets[0])
	}
	if buckets[len(buckets)-1] != budget.Seconds()*2 {
		t.Fatalf("the largest bucket ends at %v", buckets[len(buckets)-1])
	}
}
