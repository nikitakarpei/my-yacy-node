package rwidistribution

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
)

func TestOpenFansOutPostingStoredToScheduleAndLedger(t *testing.T) {
	v, err := memvault.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	now := time.Unix(1000, 0)
	distribution, err := Open(v, func() time.Time { return now })
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	word, url := yacymodel.WordHash("w1"), yacymodel.WordHash("u1")
	peer := yacymodel.WordHash("peer")
	if err := distribution.ledger.RecordAccepted(
		context.Background(),
		word,
		url,
		peer,
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	postings := fakePostingIndex{postings: map[yacymodel.Hash]yacymodel.RWIPosting{
		postingIdentity(word, url): fakePosting(word, url),
	}}
	roster := fakeRoster{responsible: []yacymodel.Seed{seed(peer)}}

	runner := distribution.Cycle(
		http.DefaultClient,
		postings,
		roster,
		fakeURLDirectory{},
		noOfferObserver{},
		Config{
			NetworkName:      "freeworld",
			Self:             yacymodel.WordHash("self"),
			Redundancy:       1,
			Partitions:       partitions,
			PostingsPerCycle: 10,
			CycleInterval:    time.Minute,
			RefreshInterval:  time.Hour,
			RetryInterval:    time.Minute,
		},
	)
	if runner == nil {
		t.Fatal("Cycle returned nil runner")
	}

	if err := distribution.schedule.Reschedule(context.Background(), word, url, now); err != nil {
		t.Fatalf("Reschedule: %v", err)
	}

	due, err := distribution.schedule.DueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("DueBatch: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word]", due)
	}
}
