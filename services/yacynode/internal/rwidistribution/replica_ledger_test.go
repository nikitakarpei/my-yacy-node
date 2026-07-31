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

func TestRecordAcceptedAddsReplicas(t *testing.T) {
	_, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peerA, peerB := yacymodel.WordHash("a"), yacymodel.WordHash("b")

	if err := ledger.RecordAccepted(context.Background(), word, url, peerA); err != nil {
		t.Fatalf("RecordAccepted a: %v", err)
	}
	if err := ledger.RecordAccepted(context.Background(), word, url, peerB); err != nil {
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
		if err := ledger.RecordAccepted(context.Background(), word, url, peer); err != nil {
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

func TestDropRemovesStaleReplicas(t *testing.T) {
	_, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	alive, dead := yacymodel.WordHash("alive"), yacymodel.WordHash("dead")

	if err := ledger.RecordAccepted(context.Background(), word, url, alive); err != nil {
		t.Fatalf("RecordAccepted alive: %v", err)
	}
	if err := ledger.RecordAccepted(context.Background(), word, url, dead); err != nil {
		t.Fatalf("RecordAccepted dead: %v", err)
	}

	dropped, err := ledger.Drop(context.Background(), word, url, []yacymodel.Hash{dead})
	if err != nil {
		t.Fatalf("Drop: %v", err)
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

func TestDropOfUnknownPostingIsHarmless(t *testing.T) {
	_, ledger := openLedger(t)

	dropped, err := ledger.Drop(
		context.Background(),
		yacymodel.WordHash("w1"),
		urlHash("u1"),
		[]yacymodel.Hash{yacymodel.WordHash("peer")},
	)
	if err != nil {
		t.Fatalf("Drop: %v", err)
	}
	if dropped != 0 {
		t.Fatalf("dropped = %v, want 0", dropped)
	}
}

func TestPostingStoredIsNoOp(t *testing.T) {
	v, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return ledger.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none", replicas)
	}
}

func TestPostingPurgedRemovesLedgerRow(t *testing.T) {
	v, ledger := openLedger(t)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")

	if err := ledger.RecordAccepted(context.Background(), word, url, peer); err != nil {
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
