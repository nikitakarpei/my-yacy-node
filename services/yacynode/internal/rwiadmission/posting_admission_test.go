package rwiadmission

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const busyPause = 5 * time.Second

type discardedHolds struct{}

func (discardedHolds) ObserveHeld(int)     {}
func (discardedHolds) ObserveReleased(int) {}
func (discardedHolds) ObserveRefused(int)  {}

type harness struct {
	index    rwipostings.PostingIndex
	escrow   *rwiescrow.HeldPostings
	urls     urlmeta.URLReceiver
	receiver PostingReceiver
}

func openHarness(t *testing.T, quotaBytes int64, batchCap int) harness {
	t.Helper()

	v, err := memvault.Open(quotaBytes)
	if err != nil {
		t.Fatalf("memvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	index, admitter, _, err := rwipostings.Open(v)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	escrow, err := rwiescrow.Open(v, admitter, discardedHolds{}, 0, time.Now)
	if err != nil {
		t.Fatalf("rwiescrow.Open: %v", err)
	}
	urlDirectory, _, urlReceiver, err := urlmeta.Open(v, escrow)
	if err != nil {
		t.Fatalf("urlmeta.Open: %v", err)
	}

	return harness{
		index:  index,
		escrow: escrow,
		urls:   urlReceiver,
		receiver: Open(
			v,
			urlDirectory,
			admitter,
			escrow,
			Config{BatchCap: batchCap, Pause: busyPause},
		),
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

func (h harness) indexed(t *testing.T, entry yacymodel.RWIPosting) bool {
	t.Helper()

	_, found, err := h.index.PostingOf(context.Background(), entry.WordHash, entry.URLHash)
	if err != nil {
		t.Fatalf("Posting: %v", err)
	}

	return found
}

func (h harness) heldCount(t *testing.T) int {
	t.Helper()

	count, err := h.escrow.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	return count
}

func (h harness) storeMetadata(t *testing.T, seeds ...string) {
	t.Helper()

	metadata := make([]yacymodel.URLMetadata, 0, len(seeds))
	for _, seed := range seeds {
		metadata = append(metadata, yacymodel.URLMetadata{Address: urlAddress(seed)})
	}
	if _, err := h.urls.Receive(context.Background(), metadata); err != nil {
		t.Fatalf("urls.Receive: %v", err)
	}
}

func TestKnownURLPostingJoinsIndex(t *testing.T) {
	h := openHarness(t, 0, 100)
	h.storeMetadata(t, "u1")
	entry := posting("w1", "u1")

	receipt, err := h.receiver.Receive(context.Background(), []yacymodel.RWIPosting{entry})
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if len(receipt.UnknownURL) != 0 {
		t.Fatalf("UnknownURL = %v, want empty", receipt.UnknownURL)
	}
	if !h.indexed(t, entry) {
		t.Fatal("posting for a known url did not reach the index")
	}
	if got := h.heldCount(t); got != 0 {
		t.Fatalf("held count = %d, want 0", got)
	}
}

func TestUnknownURLPostingWaitsInEscrow(t *testing.T) {
	h := openHarness(t, 0, 100)
	entry := posting("w1", "u1")

	receipt, err := h.receiver.Receive(context.Background(), []yacymodel.RWIPosting{entry})
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if len(receipt.UnknownURL) != 1 || receipt.UnknownURL[0] != entry.URLHash {
		t.Fatalf("UnknownURL = %v, want the referenced hash", receipt.UnknownURL)
	}
	if h.indexed(t, entry) {
		t.Fatal("posting for an unknown url reached the index")
	}
	if got := h.heldCount(t); got != 1 {
		t.Fatalf("held count = %d, want 1", got)
	}
}

func TestPostingJoinsIndexWhenItsURLMetadataArrives(t *testing.T) {
	h := openHarness(t, 0, 100)
	entry := posting("w1", "u1")

	if _, err := h.receiver.Receive(
		context.Background(),
		[]yacymodel.RWIPosting{entry},
	); err != nil {
		t.Fatalf("Receive: %v", err)
	}
	h.storeMetadata(t, "u1")

	if !h.indexed(t, entry) {
		t.Fatal("held posting did not reach the index after its url metadata arrived")
	}
	if got := h.heldCount(t); got != 0 {
		t.Fatalf("held count = %d, want 0", got)
	}
}

func TestReceiveRoutesEachPostingByItsOwnURL(t *testing.T) {
	h := openHarness(t, 0, 100)
	h.storeMetadata(t, "u1")
	known, unknown := posting("w1", "u1"), posting("w1", "u2")

	receipt, err := h.receiver.Receive(
		context.Background(),
		[]yacymodel.RWIPosting{known, unknown},
	)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if len(receipt.UnknownURL) != 1 || receipt.UnknownURL[0] != unknown.URLHash {
		t.Fatalf("UnknownURL = %v, want only the unknown hash", receipt.UnknownURL)
	}
	if !h.indexed(t, known) {
		t.Fatal("posting for a known url did not reach the index")
	}
	if h.indexed(t, unknown) {
		t.Fatal("posting for an unknown url reached the index")
	}
}

func TestReceiveBusyOverBatchCap(t *testing.T) {
	h := openHarness(t, 0, 1)

	receipt, err := h.receiver.Receive(context.Background(), []yacymodel.RWIPosting{
		posting("w1", "u1"),
		posting("w1", "u2"),
	})
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if !receipt.Busy || !receipt.TooLarge || receipt.Pause != busyPause {
		t.Fatalf("receipt = %+v, want Busy and TooLarge with the configured pause", receipt)
	}
}

func TestReceiveBusyAtCapacity(t *testing.T) {
	h := openHarness(t, 1, 100)

	if _, err := h.receiver.Receive(
		context.Background(),
		[]yacymodel.RWIPosting{posting("w1", "u1")},
	); err != nil {
		t.Fatalf("Receive: %v", err)
	}

	receipt, err := h.receiver.Receive(
		context.Background(),
		[]yacymodel.RWIPosting{posting("w2", "u2")},
	)
	if err != nil {
		t.Fatalf("Receive over capacity: %v", err)
	}
	if !receipt.Busy || receipt.Pause != busyPause {
		t.Fatalf("receipt = %+v, want Busy with the configured pause", receipt)
	}
}

func TestReceiveReportsEachUnknownURLOnce(t *testing.T) {
	h := openHarness(t, 0, 100)

	receipt, err := h.receiver.Receive(context.Background(), []yacymodel.RWIPosting{
		posting("w1", "u1"),
		posting("w2", "u1"),
	})
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if len(receipt.UnknownURL) != 1 {
		t.Fatalf("UnknownURL = %v, want one entry for the single unknown url", receipt.UnknownURL)
	}
	if got := h.heldCount(t); got != 2 {
		t.Fatalf("held count = %d, want both postings held", got)
	}
}

var _ PostingHolder = (*rwiescrow.HeldPostings)(nil)
