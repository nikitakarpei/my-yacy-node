package rwipostings

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type rwiPorts struct {
	Index    PostingIndex
	Receiver PostingReceiver
	Purger   PostingPurger
}

type harness struct {
	vault    *vault.Vault
	urls     urlmeta.URLReceiver
	rwi      rwiPorts
	observer *recordingObserver
}

func openHarness(t *testing.T, quotaBytes int64, batchCap int) harness {
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

	directory, _, urlReceiver, err := urlmeta.Open(v)
	if err != nil {
		t.Fatalf("urlmeta.Open: %v", err)
	}
	observer := &recordingObserver{}
	index, receiver, purger, err := Open(
		v,
		directory,
		Config{BatchCap: batchCap, Pause: 5 * time.Second},
		observer,
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}

	return harness{
		vault:    v,
		urls:     urlReceiver,
		rwi:      rwiPorts{Index: index, Receiver: receiver, Purger: purger},
		observer: observer,
	}
}

func urlAddress(seed string) string {
	return "http://example.com/" + seed
}

func urlHash(seed string) yacymodel.URLHash {
	hash, err := yacymodel.HashURL(urlAddress(seed))
	if err != nil {
		panic(err)
	}

	return hash
}

func posting(word, urlSeed string) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:   yacymodel.WordHash(word),
		URLHash:    urlHash(urlSeed),
		LocalLinks: 1,
		Hits:       1,
	}
}

func urlMetadata(seed string) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{Address: urlAddress(seed)}
}

func referencedHash(entry yacymodel.RWIPosting) yacymodel.Hash {
	return entry.URLHash.Hash()
}

func TestIntakePersistsAndCounts(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t, 0, 100)

	if _, err := h.urls.Receive(
		ctx,
		[]yacymodel.URLMetadata{urlMetadata("u1"), urlMetadata("u2")},
	); err != nil {
		t.Fatalf("urls.Intake: %v", err)
	}

	receipt, err := h.rwi.Receiver.Receive(ctx, []yacymodel.RWIPosting{
		posting("w1", "u1"),
		posting("w1", "u2"),
		posting("w1", "u1"),
	})
	if err != nil {
		t.Fatalf("Intake: %v", err)
	}
	if receipt.Busy || len(receipt.UnknownURL) != 0 {
		t.Fatalf("receipt = %+v, want empty", receipt)
	}

	rwiCount, err := h.rwi.Index.RWICount(ctx)
	if err != nil {
		t.Fatalf("RWICount: %v", err)
	}
	if rwiCount != 2 {
		t.Fatalf("RWICount = %d, want 2", rwiCount)
	}
}

func TestIntakeReportsUnknownURL(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t, 0, 100)
	entry := posting("w1", "u1")

	receipt, err := h.rwi.Receiver.Receive(ctx, []yacymodel.RWIPosting{entry})
	if err != nil {
		t.Fatalf("Intake: %v", err)
	}
	if len(receipt.UnknownURL) != 1 || receipt.UnknownURL[0] != referencedHash(entry) {
		t.Fatalf("UnknownURL = %v, want the referenced hash", receipt.UnknownURL)
	}

	if _, err := h.urls.Receive(ctx, []yacymodel.URLMetadata{urlMetadata("u1")}); err != nil {
		t.Fatalf("urls.Intake: %v", err)
	}

	receipt, err = h.rwi.Receiver.Receive(ctx, []yacymodel.RWIPosting{entry})
	if err != nil {
		t.Fatalf("Intake known: %v", err)
	}
	if len(receipt.UnknownURL) != 0 {
		t.Fatalf("UnknownURL = %v, want empty after url known", receipt.UnknownURL)
	}
}

func TestIntakeBusyAtCapacity(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t, 1, 100)

	receipt, err := h.rwi.Receiver.Receive(ctx, []yacymodel.RWIPosting{posting("w1", "u1")})
	if err != nil {
		t.Fatalf("Intake: %v", err)
	}
	if receipt.Busy {
		t.Fatalf("first receipt = %+v, want stored", receipt)
	}

	receipt, err = h.rwi.Receiver.Receive(ctx, []yacymodel.RWIPosting{posting("w2", "u2")})
	if err != nil {
		t.Fatalf("Intake over capacity: %v", err)
	}
	if !receipt.Busy || receipt.Pause != 5*time.Second {
		t.Fatalf("receipt = %+v, want Busy with pause 5s", receipt)
	}
}

func TestIntakeBusyOverBatchCap(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t, 0, 1)

	receipt, err := h.rwi.Receiver.Receive(ctx, []yacymodel.RWIPosting{
		posting("w1", "u1"),
		posting("w1", "u2"),
	})
	if err != nil {
		t.Fatalf("Intake: %v", err)
	}
	if !receipt.Busy {
		t.Fatalf("receipt = %+v, want Busy over batch cap", receipt)
	}
}
