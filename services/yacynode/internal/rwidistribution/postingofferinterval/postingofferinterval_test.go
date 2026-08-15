package postingofferinterval_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferinterval"
)

var testBounds = postingofferinterval.Bounds{
	Shortest: time.Minute,
	Longest:  8 * time.Minute,
}

var testDueAt = time.Unix(1_000_000, 0).UTC()

func TestWidenedIntervalDoublesFromTheShortestToTheLongest(t *testing.T) {
	for name, expected := range map[string]struct {
		previousInterval time.Duration
		widenedInterval  time.Duration
	}{
		"first miss":      {0, time.Minute},
		"second miss":     {time.Minute, 2 * time.Minute},
		"third miss":      {2 * time.Minute, 4 * time.Minute},
		"fourth miss":     {4 * time.Minute, 8 * time.Minute},
		"capped at first": {8 * time.Minute, 8 * time.Minute},
		"capped beyond":   {time.Hour, 8 * time.Minute},
	} {
		t.Run(name, func(t *testing.T) {
			widened := testBounds.WidenedFrom(expected.previousInterval)
			if widened != expected.widenedInterval {
				t.Fatalf("widened from %v = %v, want %v",
					expected.previousInterval, widened, expected.widenedInterval)
			}
		})
	}
}

func TestPauseTakesTheRequestedPauseWhenItIsLongerThanTheWidenedInterval(t *testing.T) {
	for name, expected := range map[string]struct {
		requestedPause time.Duration
		pause          time.Duration
	}{
		"no request":        {0, time.Minute},
		"shorter request":   {30 * time.Second, time.Minute},
		"longer request":    {5 * time.Minute, 5 * time.Minute},
		"beyond the cap":    {time.Hour, time.Hour},
		"equal to interval": {time.Minute, time.Minute},
	} {
		t.Run(name, func(t *testing.T) {
			if pause := testBounds.PauseFrom(0, expected.requestedPause); pause != expected.pause {
				t.Fatalf("pause with request %v = %v, want %v",
					expected.requestedPause, pause, expected.pause)
			}
		})
	}
}

func TestNextOfferIsAnchoredAtThePreviousDueTime(t *testing.T) {
	for name, expected := range map[string]struct {
		now          time.Duration
		nextOfferDue time.Duration
	}{
		"met early":            {-time.Minute, 8 * time.Minute},
		"met on time":          {0, 8 * time.Minute},
		"met late":             {3 * time.Minute, 8 * time.Minute},
		"met on the boundary":  {8 * time.Minute, 16 * time.Minute},
		"met after two misses": {17 * time.Minute, 24 * time.Minute},
	} {
		t.Run(name, func(t *testing.T) {
			now := testDueAt.Add(expected.now)

			nextOfferDue := testBounds.NextOfferDueFrom(testDueAt, now)

			if want := testDueAt.Add(expected.nextOfferDue); !nextOfferDue.Equal(want) {
				t.Fatalf("next offer due at %v, want %v", nextOfferDue, want)
			}
			if !nextOfferDue.After(now) {
				t.Fatalf("next offer due at %v, want a time after %v", nextOfferDue, now)
			}
		})
	}
}
