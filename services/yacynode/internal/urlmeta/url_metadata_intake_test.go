package urlmeta

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
)

func localIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{Hash: yacymodel.WordHash("self"), NetworkName: "freeworld"}
}

type urlPorts struct {
	Directory URLDirectory
	Evictor   URLEvictor
	Receiver  URLReceiver
}

func openModule(t *testing.T, quotaBytes int64) urlPorts {
	t.Helper()

	v, err := memvault.Open(quotaBytes)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	directory, evictor, receiver, err := Open(v)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return urlPorts{Directory: directory, Evictor: evictor, Receiver: receiver}
}

func urlMetadata(seed string) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{Address: "http://example.com/" + seed}
}

func metadataHash(t *testing.T, metadata yacymodel.URLMetadata) yacymodel.URLHash {
	t.Helper()

	hash, err := metadata.Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	return hash
}

func TestIntakePersistsAndReportsExisting(t *testing.T) {
	ctx := context.Background()
	module := openModule(t, 0)
	first := urlMetadata("a")
	second := urlMetadata("b")

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

	count, err := module.Directory.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Fatalf("Count = %d, want 2", count)
	}
}

func TestIntakeDurabilityAndLookup(t *testing.T) {
	ctx := context.Background()
	module := openModule(t, 0)
	row := urlMetadata("a")
	hash := metadataHash(t, row)

	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{row}); err != nil {
		t.Fatalf("Intake: %v", err)
	}

	rows, err := module.Directory.MetadataByHash(ctx, []yacymodel.URLHash{hash})
	if err != nil {
		t.Fatalf("RowsByHash: %v", err)
	}
	if len(rows) != 1 || metadataHash(t, rows[0]) != hash {
		t.Fatalf("RowsByHash = %v, want one matching row", rows)
	}

	missing, err := module.Directory.MissingURLs(ctx, []yacymodel.URLHash{
		hash,
		urlHash("absent"),
		urlHash("absent"),
	})
	if err != nil {
		t.Fatalf("MissingURLs: %v", err)
	}
	if len(missing) != 1 || missing[0] != urlHash("absent") {
		t.Fatalf("MissingURLs = %v, want one absent hash", missing)
	}
}

func TestIntakeBusyAtCapacity(t *testing.T) {
	ctx := context.Background()
	module := openModule(t, 1)

	receipt, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{urlMetadata("a")})
	if err != nil {
		t.Fatalf("Intake: %v", err)
	}
	if receipt.Busy {
		t.Fatalf("first receipt = %+v, want stored", receipt)
	}

	receipt, err = module.Receiver.Receive(ctx, []yacymodel.URLMetadata{urlMetadata("b")})
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
	row := urlMetadata("a")

	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{row}); err != nil {
		t.Fatalf("Intake: %v", err)
	}
	if len(observer.stored) != 1 || observer.stored[0] != metadataHash(t, row) {
		t.Fatalf("stored = %v, want one matching hash", observer.stored)
	}
}

func TestIntakeUpdatesAndNotifiesOnDuplicateURLs(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{}
	_, module := openObservedModule(t, observer)
	row := urlMetadata("a")
	hash := metadataHash(t, row)

	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{row}); err != nil {
		t.Fatalf("Intake: %v", err)
	}
	updated := row
	updated.Title = "refreshed"
	if _, err := module.Receiver.Receive(ctx, []yacymodel.URLMetadata{updated}); err != nil {
		t.Fatalf("Intake duplicate: %v", err)
	}
	if len(observer.stored) != 2 || observer.stored[1] != hash {
		t.Fatalf("stored = %v, want two matching hashes", observer.stored)
	}

	rows, err := module.Directory.MetadataByHash(ctx, []yacymodel.URLHash{hash})
	if err != nil {
		t.Fatalf("MetadataByHash: %v", err)
	}
	if len(rows) != 1 || rows[0].Title != "refreshed" {
		t.Fatalf("stored row = %+v, want refreshed title", rows)
	}
}

func TestIntakeSurvivesObserverFailure(t *testing.T) {
	ctx := context.Background()
	observer := &recordingObserver{fail: true}
	_, module := openObservedModule(t, observer)

	if _, err := module.Receiver.Receive(
		ctx,
		[]yacymodel.URLMetadata{urlMetadata("a")},
	); err != nil {
		t.Fatalf("Intake: %v", err)
	}
	count, err := module.Directory.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("Count = %d, want 1 despite observer failure", count)
	}
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}
