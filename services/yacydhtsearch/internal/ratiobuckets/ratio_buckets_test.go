package ratiobuckets_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/ratiobuckets"
)

func TestTheBucketsSpanNoneToAll(t *testing.T) {
	t.Parallel()

	buckets := ratiobuckets.Tenths()

	if buckets[0] != 0 || buckets[len(buckets)-1] != 1 {
		t.Fatalf("the buckets span %v to %v, want 0 to 1", buckets[0], buckets[len(buckets)-1])
	}
}

func TestEachBucketIsATenthWiderThanTheOneBelow(t *testing.T) {
	t.Parallel()

	buckets := ratiobuckets.Tenths()

	for index := 1; index < len(buckets); index++ {
		if width := buckets[index] - buckets[index-1]; math.Abs(width-0.1) > 1e-9 {
			t.Fatalf("bucket %v follows %v", buckets[index], buckets[index-1])
		}
	}
}
