package urlmeta_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

func localIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{
		Hash:         yacymodel.WordHash("self"),
		NetworkName:  "freeworld",
		Capabilities: yacymodel.PeerCapabilities{AcceptRemoteIndex: true},
	}
}

type urlPorts struct {
	Directory urlmeta.URLDirectory
	Evictor   urlmeta.URLEvictor
	Receiver  urlmeta.URLReceiver
}

func openModule(t *testing.T, quotaBytes int64) (*vault.Vault, urlPorts) {
	t.Helper()

	v, err := vault.New(
		vaultenginetest.EngineRepeatingWrites(memoryvault.OpenEngine(quotaBytes)),
		nil,
	)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	directory, evictor, _, receiver, err := urlmeta.Open(context.Background(), v, intakeClock, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return v, urlPorts{Directory: directory, Evictor: evictor, Receiver: receiver}
}

func intakeClock() time.Time {
	return time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)
}

func urlMetadata(t *testing.T, seed string) yacymodel.URLMetadata {
	t.Helper()

	address := "http://example.com/" + seed

	return yacymodel.URLMetadata{Hash: urlHashOf(t, address), Address: address}
}

func urlHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return hash
}

func TestIntakePersistsAndReportsExisting(t *testing.T) {
	ctx := context.Background()
	v, module := openModule(t, 0)
	first := urlMetadata(t, "a")
	second := urlMetadata(t, "b")

	receipt, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{first, second})
	if err != nil {
		t.Fatalf("Intake: %v", err)
	}
	if receipt.Busy || receipt.Double != 0 || len(receipt.ErrorURL) != 0 {
		t.Fatalf("first receipt = %+v, want empty", receipt)
	}

	receipt, err = module.Receiver.Receive(ctx, []yacymodel.URLMetadata{first})
	if err != nil {
		t.Fatalf("Intake duplicate: %v", err)
	}
	if receipt.Double != 1 {
		t.Fatalf("duplicate Double = %d, want 1", receipt.Double)
	}

	if count := storedURLCount(t, v, module.Directory); count != 2 {
		t.Fatalf("Count = %d, want 2", count)
	}
}

func TestIntakeDurabilityAndLookup(t *testing.T) {
	ctx := context.Background()
	v, module := openObservedModule(t)
	row := urlMetadata(t, "a")
	hash := row.Hash

	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{row}); err != nil {
		t.Fatalf("Intake: %v", err)
	}

	rows := metadataPerHash(t, v, module.Directory, []yacymodel.URLHash{hash, urlHash("absent")})
	if len(rows) != 1 || rows[hash].Hash != hash {
		t.Fatalf("MetadataPerHash = %v, want one matching row and no absent one", rows)
	}
}

func TestIntakeBusyAtCapacity(t *testing.T) {
	ctx := context.Background()
	_, module := openModule(t, 1)

	receipt, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{urlMetadata(t, "a")})
	if err != nil {
		t.Fatalf("Intake: %v", err)
	}
	if receipt.Busy {
		t.Fatalf("first receipt = %+v, want stored", receipt)
	}

	receipt, err = module.Receiver.Receive(ctx, []yacymodel.URLMetadata{urlMetadata(t, "b")})
	if err != nil {
		t.Fatalf("Intake over capacity: %v", err)
	}
	if !receipt.Busy {
		t.Fatalf("receipt = %+v, want Busy", receipt)
	}
}

func TestIntakeNotifiesObserverOfStoredURLs(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{}
	_, module := openObservedModule(t, observer)
	row := urlMetadata(t, "a")

	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{row}); err != nil {
		t.Fatalf("Intake: %v", err)
	}
	if len(observer.stored) != 1 || observer.stored[0] != row.Hash {
		t.Fatalf("stored = %v, want one matching hash", observer.stored)
	}
}

func TestIntakeUpdatesAndNotifiesOnALaterVersionOfAURL(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{}
	v, module := openObservedModule(t, observer)
	row := urlMetadata(t, "a")
	row.Modified = yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, 1))
	hash := row.Hash

	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{row}); err != nil {
		t.Fatalf("Intake: %v", err)
	}
	updated := row
	updated.Modified = yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, 2))
	updated.Title = "refreshed"
	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{updated}); err != nil {
		t.Fatalf("Intake duplicate: %v", err)
	}
	if len(observer.stored) != 2 || observer.stored[1] != hash {
		t.Fatalf("stored = %v, want two matching hashes", observer.stored)
	}

	rows := metadataPerHash(t, v, module.Directory, []yacymodel.URLHash{hash})
	if len(rows) != 1 || rows[hash].Title != "refreshed" {
		t.Fatalf("stored row = %+v, want refreshed title", rows)
	}
}

func TestIntakeKeepsHeldMetadataOverAnOlderVersion(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{}
	v, module := openObservedModule(t, observer)
	held := urlMetadata(t, "a")
	held.Modified = yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, 10))
	held.Title = "held"
	older := held
	older.Modified = yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, 1))
	older.Title = "older"

	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{held}); err != nil {
		t.Fatalf("Intake: %v", err)
	}
	receipt, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{older})
	if err != nil {
		t.Fatalf("Intake older: %v", err)
	}

	if receipt.Double != 1 {
		t.Fatalf("double = %d, want the older metadata counted as double", receipt.Double)
	}
	rows := metadataPerHash(t, v, module.Directory, []yacymodel.URLHash{held.Hash})
	if rows[held.Hash].Title != "held" {
		t.Fatalf("stored title = %q, want the held metadata kept", rows[held.Hash].Title)
	}
	if len(observer.storedFingerprints) != 1 ||
		observer.storedFingerprints[0] != pagefingerprint.FromMetadata(held) {
		t.Fatalf(
			"stored fingerprints = %v, want only the fingerprint of the held metadata",
			observer.storedFingerprints,
		)
	}
}

func TestIntakeSkipsMetadataDatedAfterToday(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{}
	v, module := openObservedModule(t, observer)
	postdated := urlMetadata(t, "a")
	postdated.Modified = yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, 17))

	receipt, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{postdated})
	if err != nil {
		t.Fatalf("Intake: %v", err)
	}

	rows := metadataPerHash(t, v, module.Directory, []yacymodel.URLHash{postdated.Hash})
	if len(rows) != 0 || receipt.Double != 0 || len(observer.storedFingerprints) != 0 {
		t.Fatalf(
			"stored = %+v, double = %d, observed = %v, want metadata dated tomorrow skipped",
			rows, receipt.Double, observer.storedFingerprints,
		)
	}
}

func TestIntakeKeepsStoredMetadataOverAnotherVersionOfTheSameDay(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{}
	v, module := openObservedModule(t, observer)
	stored := urlMetadata(t, "a")
	stored.Modified = yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, 10))
	stored.Title = "stored"
	sameDay := stored
	sameDay.LocalLinks = 9
	sameDay.Title = "same day"

	for _, metadata := range []yacymodel.URLMetadata{stored, sameDay} {
		if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{metadata}); err != nil {
			t.Fatalf("Intake: %v", err)
		}
	}

	rows := metadataPerHash(t, v, module.Directory, []yacymodel.URLHash{stored.Hash})
	if rows[stored.Hash].Title != "stored" || len(observer.stored) != 1 {
		t.Fatalf(
			"stored title = %q, observed %d stores, want the first metadata of the day kept",
			rows[stored.Hash].Title, len(observer.stored),
		)
	}
}

func TestIntakeStoresNothingWhenAnObserverFails(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{fail: true}
	v, module := openObservedModule(t, observer)

	if _, err := module.Receiver.Receive(
		ctx,
		[]yacymodel.URLMetadata{urlMetadata(t, "a")},
	); err == nil {
		t.Fatal("Intake accepted the url although an observer refused it")
	}
	if count := storedURLCount(t, v, module.Directory); count != 0 {
		t.Fatalf("Count = %d, want the refused url left unstored", count)
	}
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}
