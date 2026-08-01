package rwidistribution

import (
	"testing"
	"time"
)

func TestNextDueUsesRefreshWhenReplicated(t *testing.T) {
	cadence := postingOfferCadence{refresh: time.Hour, retry: time.Minute}
	now := time.Unix(1000, 0)

	got := cadence.NextDue(now, true, 0)
	if want := now.Add(time.Hour); !got.Equal(want) {
		t.Fatalf("NextDue = %v, want %v", got, want)
	}
}

func TestNextDueUsesRetryAfterWhenGiven(t *testing.T) {
	cadence := postingOfferCadence{refresh: time.Hour, retry: time.Minute}
	now := time.Unix(1000, 0)

	got := cadence.NextDue(now, false, 5*time.Minute)
	if want := now.Add(5 * time.Minute); !got.Equal(want) {
		t.Fatalf("NextDue = %v, want %v", got, want)
	}
}

func TestNextDueFallsBackToRetryWithoutRetryAfter(t *testing.T) {
	cadence := postingOfferCadence{refresh: time.Hour, retry: time.Minute}
	now := time.Unix(1000, 0)

	got := cadence.NextDue(now, false, 0)
	if want := now.Add(time.Minute); !got.Equal(want) {
		t.Fatalf("NextDue = %v, want %v", got, want)
	}
}
