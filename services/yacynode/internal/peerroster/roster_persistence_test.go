package peerroster_test

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

func TestRosterRestoresPeerAddressesWithoutRestoringReachability(t *testing.T) {
	storagePath := filepath.Join(t.TempDir(), "peer-roster")
	peerSeed := seniorSeed(t, "peer", "203.0.113.10", 8090)
	additionalHost, err := yacymodel.ParseHost("peer.example")
	if err != nil {
		t.Fatalf("parse host: %v", err)
	}
	peerSeed.AdditionalAddresses = yacymodel.Some([]yacymodel.Host{additionalHost})
	peerSeed.SecurePort = yacymodel.Some(yacymodel.Port(8443))
	peerSeed.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{SSLAvailable: true})

	storedVault := openRosterVault(t, storagePath)
	storedRoster := openRoster(
		t,
		rosterFixture{storage: storedVault, reservoirCap: 8, reachableCap: 4},
	)
	storedRoster.Discover(t.Context(), peerSeed)
	if err := storedVault.Close(); err != nil {
		t.Fatalf("Close stored roster: %v", err)
	}

	restoredVault := openRosterVault(t, storagePath)
	t.Cleanup(func() {
		if err := restoredVault.Close(); err != nil {
			t.Errorf("Close restored roster: %v", err)
		}
	})
	restoredRoster := openRoster(
		t,
		rosterFixture{storage: restoredVault, reservoirCap: 8, reachableCap: 4},
	)
	if got := restoredRoster.ReachablePeers(t.Context()); len(got) != 0 {
		t.Fatalf("reachable peers after restart = %d, want 0", len(got))
	}
	if _, found := hashSet(restoredRoster.UnreachablePeerHashes(t.Context(), 1))[peerSeed.Hash]; !found {
		t.Fatalf("persisted peer missing from unreachable peer hashes")
	}

	restoredAddresses, found := restoredRoster.AddressesOf(t.Context(), peerSeed.Hash)
	if !found {
		t.Fatal("persisted peer addresses not found")
	}
	wantAddresses, _ := peerSeed.Addresses()
	if !reflect.DeepEqual(restoredAddresses, wantAddresses) {
		t.Errorf("addresses = %v, want %v", restoredAddresses, wantAddresses)
	}
}

func TestRosterRestoresTheAnsweredOrigin(t *testing.T) {
	storagePath := filepath.Join(t.TempDir(), "peer-roster")
	peerSeed := seniorSeed(t, "peer", "203.0.113.10", 8090)
	answeredOrigin := httpOriginOf(peerSeed)
	answeredOrigin.Scheme = peerwire.HTTPS
	answeredOrigin.Port = 443

	storedVault := openRosterVault(t, storagePath)
	storedRoster := openRoster(
		t,
		rosterFixture{storage: storedVault, reservoirCap: 8, reachableCap: 4},
	)
	storedRoster.ConfirmReachable(t.Context(), peerSeed, answeredOrigin)
	if err := storedVault.Close(); err != nil {
		t.Fatalf("Close stored roster: %v", err)
	}

	restoredVault := openRosterVault(t, storagePath)
	t.Cleanup(func() {
		if err := restoredVault.Close(); err != nil {
			t.Errorf("Close restored roster: %v", err)
		}
	})
	restoredRoster := openRoster(
		t,
		rosterFixture{storage: restoredVault, reservoirCap: 8, reachableCap: 4},
	)

	if origin, found := restoredRoster.AnsweredOriginOf(t.Context(), peerSeed.Hash); !found ||
		origin != answeredOrigin {
		t.Fatalf("answered origin = %v, %v, want %v", origin, found, answeredOrigin)
	}
}

func openRosterVault(t testing.TB, storagePath string) *vault.Vault {
	t.Helper()

	engine, err := pebblevault.OpenEngine(storagePath, 0, pebblevault.MachineLimits{}, nil, nil)
	if err != nil {
		t.Fatalf("pebblevault.OpenEngine: %v", err)
	}
	storage, err := vault.New(engine, nil, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	return storage
}

type recordBytesCodec struct{}

func (recordBytesCodec) Encode(record []byte) ([]byte, error) { return record, nil }

func (recordBytesCodec) Decode(raw []byte) ([]byte, error) { return raw, nil }

func TestRosterReadsARecordStoredBeforeAdditionalAddresses(t *testing.T) {
	storagePath := filepath.Join(t.TempDir(), "peer-roster")
	peerSeed := seniorSeed(t, "peer", "203.0.113.10", 8090)
	var record storedfields.Writer
	record.Time(time.Time{})
	record.Time(time.Time{})
	record.Text("203.0.113.10")
	record.Varint(8090)

	storedVault := openRosterVault(t, storagePath)
	records, err := storedVault.RegisterCollection(
		vault.Name("peerroster"),
		vault.SingleKey(hashkeypart.Hash).KeyLayout(),
		recordBytesCodec{},
	)
	if err != nil {
		t.Fatalf("register roster records: %v", err)
	}
	if err := storedVault.Update(t.Context(), func(tx *vault.Txn) error {
		_, err := records.Put(tx, peerSeed.Hash, record.Record())

		return err
	}); err != nil {
		t.Fatalf("store record: %v", err)
	}
	if err := storedVault.Close(); err != nil {
		t.Fatalf("Close stored roster: %v", err)
	}

	restoredVault := openRosterVault(t, storagePath)
	t.Cleanup(func() {
		if err := restoredVault.Close(); err != nil {
			t.Errorf("Close restored roster: %v", err)
		}
	})
	restoredRoster := openRoster(
		t,
		rosterFixture{storage: restoredVault, reservoirCap: 8, reachableCap: 4},
	)

	addresses, found := restoredRoster.AddressesOf(t.Context(), peerSeed.Hash)
	wantAddresses, _ := peerSeed.Addresses()
	if !found || !reflect.DeepEqual(addresses, wantAddresses) {
		t.Errorf("addresses = %v, %v, want %v", addresses, found, wantAddresses)
	}
}
