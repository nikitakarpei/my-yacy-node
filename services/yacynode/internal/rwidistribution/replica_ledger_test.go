package rwidistribution

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

func openLedger(t *testing.T) (*vault.Vault, *replicaLedger) {
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

	ledger, err := openReplicaLedger(v)
	if err != nil {
		t.Fatalf("openReplicaLedger: %v", err)
	}

	return v, ledger
}

func acceptedOffer(peer yacymodel.Hash, word yacymodel.Hash, url yacymodel.URLHash) postingOffer {
	return postingOffer{
		Peer:     seed(peer),
		Postings: []yacymodel.RWIPosting{fakePosting(word, url)},
	}
}

func TestRecordAcceptedAddsReplicas(t *testing.T) {
	_, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peerA, peerB := yacymodel.WordHash("a"), yacymodel.WordHash("b")

	if err := ledger.RecordAccepted(
		context.Background(),
		acceptedOffer(peerA, word, url),
	); err != nil {
		t.Fatalf("RecordAccepted a: %v", err)
	}
	if err := ledger.RecordAccepted(
		context.Background(),
		acceptedOffer(peerB, word, url),
	); err != nil {
		t.Fatalf("RecordAccepted b: %v", err)
	}

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 2 {
		t.Fatalf("replicas = %v, want 2 entries", replicas)
	}
}

func TestRecordAcceptedIsIdempotent(t *testing.T) {
	_, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("a")

	for range 2 {
		if err := ledger.RecordAccepted(
			context.Background(),
			acceptedOffer(peer, word, url),
		); err != nil {
			t.Fatalf("RecordAccepted: %v", err)
		}
	}

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 1 {
		t.Fatalf("replicas = %v, want 1 entry", replicas)
	}
}

func TestRecordAcceptedCoversAllPostingsInOffer(t *testing.T) {
	_, ledger := openLedger(t)
	word := yacymodel.WordHash("w1")
	urlA, urlB := urlHash("u1"), urlHash("u2")
	peer := yacymodel.WordHash("a")

	offer := postingOffer{
		Peer: seed(peer),
		Postings: []yacymodel.RWIPosting{
			fakePosting(word, urlA),
			fakePosting(word, urlB),
		},
	}
	if err := ledger.RecordAccepted(context.Background(), offer); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	for _, url := range []yacymodel.URLHash{urlA, urlB} {
		replicas, err := ledger.Replicas(context.Background(), word, url)
		if err != nil {
			t.Fatalf("Replicas: %v", err)
		}
		if len(replicas) != 1 || replicas[0] != peer {
			t.Fatalf("replicas for %v = %v, want [%v]", url, replicas, peer)
		}
	}
}

func TestRecordDroppedRemovesStaleReplicas(t *testing.T) {
	_, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	alive, dead := yacymodel.WordHash("alive"), yacymodel.WordHash("dead")

	if err := ledger.RecordAccepted(
		context.Background(),
		acceptedOffer(alive, word, url),
	); err != nil {
		t.Fatalf("RecordAccepted alive: %v", err)
	}
	if err := ledger.RecordAccepted(
		context.Background(),
		acceptedOffer(dead, word, url),
	); err != nil {
		t.Fatalf("RecordAccepted dead: %v", err)
	}

	dropped, err := ledger.RecordDropped(context.Background(), word, url, []yacymodel.Hash{dead})
	if err != nil {
		t.Fatalf("RecordDropped: %v", err)
	}
	if dropped != 1 {
		t.Fatalf("dropped = %v, want 1", dropped)
	}

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 1 || replicas[0] != alive {
		t.Fatalf("replicas = %v, want [alive]", replicas)
	}
}

func TestRecordDroppedOfUnknownPostingIsHarmless(t *testing.T) {
	_, ledger := openLedger(t)

	dropped, err := ledger.RecordDropped(
		context.Background(),
		yacymodel.WordHash("w1"),
		urlHash("u1"),
		[]yacymodel.Hash{yacymodel.WordHash("peer")},
	)
	if err != nil {
		t.Fatalf("RecordDropped: %v", err)
	}
	if dropped != 0 {
		t.Fatalf("dropped = %v, want 0", dropped)
	}
}

func TestPostingPurgedRemovesLedgerRow(t *testing.T) {
	v, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")

	if err := ledger.RecordAccepted(
		context.Background(),
		acceptedOffer(peer, word, url),
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return ledger.PostingPurged(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none after purge", replicas)
	}
}
