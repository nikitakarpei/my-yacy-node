package postingreplicas_test

import (
	"context"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
)

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func testIdentity() postingidentity.Identity {
	return postingidentity.Identity{Word: yacymodel.WordHash("w1"), URL: urlHash("u1")}
}

func recordHolders(
	t *testing.T,
	v *vault.Vault,
	ledger *postingreplicas.Replicas,
	identity postingidentity.Identity,
	holders ...yacymodel.Hash,
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		recordedHolders, err := ledger.HoldersOf(tx, identity)
		if err != nil {
			return err
		}

		return ledger.RecordHolders(tx, identity, recordedHolders, holders)
	}); err != nil {
		t.Fatalf("RecordHolders: %v", err)
	}
}

func holdersOf(
	t *testing.T,
	v *vault.Vault,
	ledger *postingreplicas.Replicas,
	identity postingidentity.Identity,
) []yacymodel.Hash {
	t.Helper()

	var holders []yacymodel.Hash
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		holders, err = ledger.HoldersOf(tx, identity)

		return err
	}); err != nil {
		t.Fatalf("HoldersOf: %v", err)
	}

	return holders
}

func openLedger(t *testing.T) (*vault.Vault, *postingreplicas.Replicas) {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	ledger, err := postingreplicas.Open(v)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return v, ledger
}

func TestRecordHoldersStoresTheHolders(t *testing.T) {
	v, ledger := openLedger(t)
	peerA, peerB := yacymodel.WordHash("a"), yacymodel.WordHash("b")

	recordHolders(t, v, ledger, testIdentity(), peerA, peerB)

	holders := holdersOf(t, v, ledger, testIdentity())
	if !slices.Equal(holders, []yacymodel.Hash{peerA, peerB}) {
		t.Fatalf("holders = %v, want [a b]", holders)
	}
}

func TestRecordHoldersKeepsUnchangedHolders(t *testing.T) {
	v, ledger := openLedger(t)
	peer := yacymodel.WordHash("a")
	recordHolders(t, v, ledger, testIdentity(), peer)

	recordHolders(t, v, ledger, testIdentity(), peer)

	holders := holdersOf(t, v, ledger, testIdentity())
	if !slices.Equal(holders, []yacymodel.Hash{peer}) {
		t.Fatalf("holders = %v, want [a]", holders)
	}
}

func TestRecordHoldersWithoutPeersDropsTheLedgerRow(t *testing.T) {
	v, ledger := openLedger(t)
	recordHolders(t, v, ledger, testIdentity(), yacymodel.WordHash("a"))

	recordHolders(t, v, ledger, testIdentity())

	if holders := holdersOf(t, v, ledger, testIdentity()); len(holders) != 0 {
		t.Fatalf("holders = %v, want none", holders)
	}
}

func TestPostingPurgedRemovesLedgerRow(t *testing.T) {
	v, ledger := openLedger(t)
	identity := testIdentity()
	recordHolders(t, v, ledger, identity, yacymodel.WordHash("peer"))

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return ledger.PostingPurged(
			tx, yacymodel.RWIPosting{WordHash: identity.Word, URLHash: identity.URL},
		)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}

	if holders := holdersOf(t, v, ledger, identity); len(holders) != 0 {
		t.Fatalf("holders = %v, want none after purge", holders)
	}
}
