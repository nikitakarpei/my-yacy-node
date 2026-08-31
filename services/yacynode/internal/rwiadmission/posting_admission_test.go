package rwiadmission_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const busyPause = 5 * time.Second

type discardedHolds struct{}

func (discardedHolds) ObserveHeld(int)     {}
func (discardedHolds) ObserveReleased(int) {}

type recordedRefusals struct {
	postings map[rwiadmission.RefusalReason]int
}

func (r *recordedRefusals) ObserveRefused(reason rwiadmission.RefusalReason, postings int) {
	r.postings[reason] += postings
}

type harness struct {
	vault    *vault.Vault
	index    rwipostings.PostingIndex
	escrow   *rwiescrow.PostingEscrow
	urls     urlmeta.URLReceiver
	receiver rwiadmission.PostingReceiver
	refusals *recordedRefusals
}

func openHarness(t *testing.T, quotaBytes int64, escrowCapacity int) harness {
	t.Helper()

	v, err := memoryvault.Open(quotaBytes, nil)
	if err != nil {
		t.Fatalf("memoryvault.Open: %v", err)
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
	escrow, err := rwiescrow.Open(v, admitter, discardedHolds{}, escrowCapacity, time.Now)
	if err != nil {
		t.Fatalf("rwiescrow.Open: %v", err)
	}
	urlDirectory, _, urlReceiver, err := urlmeta.Open(v, escrow)
	if err != nil {
		t.Fatalf("urlmeta.Open: %v", err)
	}
	refusals := &recordedRefusals{postings: map[rwiadmission.RefusalReason]int{}}

	return harness{
		vault:  v,
		index:  index,
		escrow: escrow,
		urls:   urlReceiver,
		receiver: rwiadmission.Open(
			v,
			urlDirectory,
			admitter,
			escrow,
			rwiadmission.Config{Pause: busyPause, Refusals: refusals},
		),
		refusals: refusals,
	}
}

func urlAddress(seed string) string {
	return "http://example.com/" + seed
}

func urlHash(seed string) yacymodel.URLHash {
	address, err := url.Parse(urlAddress(seed))
	if err != nil {
		panic(err)
	}

	return yacymodel.URLNormalformOf(address).Hash()
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

	var count int
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		measured, err := h.escrow.Count(tx)
		count = measured

		return err
	}); err != nil {
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
	if got := h.refusals.postings[rwiadmission.RefusalStorageFull]; got != 1 {
		t.Fatalf("postings refused for a full storage = %d, want the whole request of 1", got)
	}
}

func TestReceiveBusyWhenTheEscrowIsFull(t *testing.T) {
	h := openHarness(t, 0, 1)

	if _, err := h.receiver.Receive(
		context.Background(),
		[]yacymodel.RWIPosting{posting("w1", "u1")},
	); err != nil {
		t.Fatalf("Receive: %v", err)
	}

	refused := []yacymodel.RWIPosting{posting("w2", "u2"), posting("w3", "u3")}
	receipt, err := h.receiver.Receive(context.Background(), refused)
	if err != nil {
		t.Fatalf("Receive over the escrow capacity: %v", err)
	}
	if !receipt.Busy || receipt.Pause != busyPause {
		t.Fatalf("receipt = %+v, want Busy with the configured pause", receipt)
	}
	if got := h.heldCount(t); got != 1 {
		t.Fatalf("held count = %d, want the refused request to leave no posting behind", got)
	}
	if got := h.refusals.postings[rwiadmission.RefusalEscrowFull]; got != len(refused) {
		t.Fatalf("postings refused for a full escrow = %d, want the whole request of %d",
			got, len(refused))
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

var _ rwiadmission.PostingHolder = (*rwiescrow.PostingEscrow)(nil)
