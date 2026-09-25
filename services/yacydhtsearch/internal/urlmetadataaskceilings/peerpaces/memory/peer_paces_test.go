package memory_test

import (
	"sync"
	"testing"
	"time"

	peerpacesmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/urlmetadataaskceilings/peerpaces/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func setTo(
	pace time.Duration,
) func(yacymodel.Optional[time.Duration]) time.Duration {
	return func(yacymodel.Optional[time.Duration]) time.Duration { return pace }
}

func oneMillisecondLonger(pace yacymodel.Optional[time.Duration]) time.Duration {
	return pace.OrElse(0) + time.Millisecond
}

func TestAnUpdatedTimeIsReturnedForItsAddressOnly(t *testing.T) {
	t.Parallel()

	paces := peerpacesmemory.New()
	paces.Update(t.Context(), "http://first.example", setTo(2*time.Millisecond))

	if pace, known := paces.Read(
		t.Context(), "http://first.example",
	).Get(); !known || pace != 2*time.Millisecond {
		t.Fatalf("pace of the updated address = %v (known %v), want 2ms",
			pace, known)
	}
	if paces.Read(t.Context(), "http://second.example").Present() {
		t.Fatal("an address never updated has a pace")
	}
}

func TestAnUpdateStartsFromTheTimeBeforeItAndReturnsTheNewOne(t *testing.T) {
	t.Parallel()

	paces := peerpacesmemory.New()
	first := paces.Update(t.Context(), "http://first.example", oneMillisecondLonger)
	second := paces.Update(t.Context(), "http://first.example", oneMillisecondLonger)

	if first != time.Millisecond || second != 2*time.Millisecond {
		t.Fatalf("two updates returned %v and %v, want 1ms and 2ms", first, second)
	}
}

func TestConcurrentUpdatesOfOneAddressLoseNone(t *testing.T) {
	t.Parallel()

	paces := peerpacesmemory.New()
	var updates sync.WaitGroup
	for range 100 {
		updates.Go(
			func() { paces.Update(t.Context(), "http://first.example", oneMillisecondLonger) },
		)
	}
	updates.Wait()

	if pace, _ := paces.Read(
		t.Context(), "http://first.example",
	).Get(); pace != 100*time.Millisecond {
		t.Fatalf("pace after 100 concurrent updates = %v, want 100ms",
			pace)
	}
}
