package postingreplicas

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func fakePosting(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{WordHash: word, URLHash: url}
}

func store(
	t *testing.T,
	v *vault.Vault,
	schedule *postingofferschedule.Schedule,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
}

func recordAccepted(
	t *testing.T,
	v *vault.Vault,
	ledger *Replicas,
	peer yacymodel.Hash,
	postings ...yacymodel.RWIPosting,
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return ledger.RecordAccepted(tx, peer, postings)
	}); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}
}

func recordDropped(
	t *testing.T,
	v *vault.Vault,
	ledger *Replicas,
	posting postingidentity.Identity,
	stale []yacymodel.Hash,
) int {
	t.Helper()

	var dropped int
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		var err error
		dropped, err = ledger.DropStaleHolders(
			tx,
			map[postingidentity.Identity][]yacymodel.Hash{posting: stale},
		)

		return err
	}); err != nil {
		t.Fatalf("DropStaleHolders: %v", err)
	}

	return dropped
}

func holdersOf(
	t *testing.T,
	v *vault.Vault,
	ledger *Replicas,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) []yacymodel.Hash {
	t.Helper()

	var holders []yacymodel.Hash
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		holders, err = ledger.HoldersOf(tx, postingidentity.IdentityOf(word, url))

		return err
	}); err != nil {
		t.Fatalf("Holders: %v", err)
	}

	return holders
}

func openLedger(t *testing.T) (*vault.Vault, *Replicas) {
	t.Helper()

	v, err := memvault.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	schedule, err := postingofferschedule.Open(v, time.Now, discardedScheduleObservations{})
	if err != nil {
		t.Fatalf("postingofferschedule.Open: %v", err)
	}

	ledger, err := Open(v, schedule)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	for _, posting := range []struct {
		word yacymodel.Hash
		url  yacymodel.URLHash
	}{
		{yacymodel.WordHash("w1"), urlHash("u1")},
		{yacymodel.WordHash("w1"), urlHash("u2")},
	} {
		store(t, v, schedule, posting.word, posting.url)
	}

	return v, ledger
}

func TestRecordAcceptedAddsReplicas(t *testing.T) {
	v, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peerA, peerB := yacymodel.WordHash("a"), yacymodel.WordHash("b")

	recordAccepted(t, v, ledger, peerA, fakePosting(word, url))
	recordAccepted(t, v, ledger, peerB, fakePosting(word, url))

	holders := holdersOf(t, v, ledger, word, url)
	if len(holders) != 2 {
		t.Fatalf("holders = %v, want 2 entries", holders)
	}
}

func TestRecordAcceptedIsIdempotent(t *testing.T) {
	v, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("a")

	for range 2 {
		recordAccepted(t, v, ledger, peer, fakePosting(word, url))
	}

	holders := holdersOf(t, v, ledger, word, url)
	if len(holders) != 1 {
		t.Fatalf("holders = %v, want 1 entry", holders)
	}
}

func TestRecordAcceptedCoversAllPostingsInOffer(t *testing.T) {
	v, ledger := openLedger(t)
	word := yacymodel.WordHash("w1")
	urlA, urlB := urlHash("u1"), urlHash("u2")
	peer := yacymodel.WordHash("a")

	postings := []yacymodel.RWIPosting{
		fakePosting(word, urlA),
		fakePosting(word, urlB),
	}
	recordAccepted(t, v, ledger, peer, postings...)

	for _, url := range []yacymodel.URLHash{urlA, urlB} {
		holders := holdersOf(t, v, ledger, word, url)
		if len(holders) != 1 || holders[0] != peer {
			t.Fatalf("holders for %v = %v, want [%v]", url, holders, peer)
		}
	}
}

func TestRecordAcceptedSkipsPostingWithNoDueRow(t *testing.T) {
	v, ledger := openLedger(t)
	word, url := yacymodel.WordHash("absent"), urlHash("absent")
	peer := yacymodel.WordHash("a")

	recordAccepted(t, v, ledger, peer, fakePosting(word, url))

	holders := holdersOf(t, v, ledger, word, url)
	if len(holders) != 0 {
		t.Fatalf("holders = %v, want none for a posting with no due row", holders)
	}
}

func TestRecordDroppedRemovesStaleReplicas(t *testing.T) {
	v, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	alive, dead := yacymodel.WordHash("alive"), yacymodel.WordHash("dead")

	recordAccepted(t, v, ledger, alive, fakePosting(word, url))
	recordAccepted(t, v, ledger, dead, fakePosting(word, url))

	dropped := recordDropped(
		t,
		v,
		ledger,
		postingidentity.Identity{Word: word, URL: url},
		[]yacymodel.Hash{dead},
	)
	if dropped != 1 {
		t.Fatalf("dropped = %v, want 1", dropped)
	}

	holders := holdersOf(t, v, ledger, word, url)
	if len(holders) != 1 || holders[0] != alive {
		t.Fatalf("holders = %v, want [alive]", holders)
	}
}

func TestRecordDroppedOfUnknownPostingIsHarmless(t *testing.T) {
	v, ledger := openLedger(t)

	dropped := recordDropped(
		t, v, ledger,
		postingidentity.Identity{Word: yacymodel.WordHash("w1"), URL: urlHash("u1")},
		[]yacymodel.Hash{yacymodel.WordHash("peer")},
	)
	if dropped != 0 {
		t.Fatalf("dropped = %v, want 0", dropped)
	}
}

func TestPostingPurgedRemovesLedgerRow(t *testing.T) {
	v, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")

	recordAccepted(t, v, ledger, peer, fakePosting(word, url))

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return ledger.PostingPurged(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}

	holders := holdersOf(t, v, ledger, word, url)
	if len(holders) != 0 {
		t.Fatalf("holders = %v, want none after purge", holders)
	}
}

type discardedScheduleObservations struct{}

func (discardedScheduleObservations) ObserveScheduledPostings(int) {}

func (discardedScheduleObservations) ObserveLongestOfferLateness(time.Duration) {}
