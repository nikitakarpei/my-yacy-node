package rwiadmission_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

const busyPause = 5 * time.Second

type discardedHolds struct{}

func (discardedHolds) ObserveHeld(*vault.Txn, int)     {}
func (discardedHolds) ObserveReleased(*vault.Txn, int) {}

type recordedRefusals struct {
	postings map[rwiadmission.RefusalReason]int
}

func (r *recordedRefusals) ObserveRefused(reason rwiadmission.RefusalReason, postings int) {
	r.postings[reason] += postings
}

type harness struct {
	vault        *vault.Vault
	index        rwipostings.PostingIndex
	escrow       *rwiescrow.PostingEscrow
	urls         urlmeta.URLReceiver
	receiver     rwiadmission.PostingReceiver
	pageReceiver rwiadmission.PagePostingReceiver
	refusals     *recordedRefusals
}

type refusedAdmitter struct {
	admitter rwipostings.PostingAdmitter
	word     yacymodel.Hash
}

func (r refusedAdmitter) Admit(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if posting.WordHash == r.word {
		return errAdmissionRefused
	}

	return r.admitter.Admit(tx, posting)
}

var errAdmissionRefused = errors.New("admission refused")

func openHarness(t *testing.T, quotaBytes int64, escrowCapacity int) harness {
	t.Helper()

	return openHarnessAdmitting(t, quotaBytes, escrowCapacity, nil)
}

func openHarnessAdmitting(
	t *testing.T,
	quotaBytes int64,
	escrowCapacity int,
	admittedThrough func(rwipostings.PostingAdmitter) rwipostings.PostingAdmitter,
) harness {
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

	references, err := urlreferences.Open(v)
	if err != nil {
		t.Fatalf("urlreferences.Open: %v", err)
	}
	index, admitter, purger, err := rwipostings.Open(v, references)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	if admittedThrough != nil {
		admitter = admittedThrough(admitter)
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
	receiver, pageReceiver := rwiadmission.Open(
		v,
		urlDirectory,
		admitter,
		purger,
		references,
		escrow,
		rwiadmission.Config{Pause: busyPause, Refusals: refusals},
	)

	return harness{
		vault:        v,
		index:        index,
		escrow:       escrow,
		urls:         urlReceiver,
		receiver:     receiver,
		pageReceiver: pageReceiver,
		refusals:     refusals,
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
		Language:   yacymodel.LanguageOfUndeclaredDocument,
		LocalLinks: 1,
		Hits:       1,
	}
}

func (h harness) indexed(t *testing.T, entry yacymodel.RWIPosting) bool {
	t.Helper()

	var found bool
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		_, stored, err := h.index.PostingOf(tx, entry.WordHash, entry.URLHash)
		found = stored

		return err
	}); err != nil {
		t.Fatalf("PostingOf: %v", err)
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
		address := urlAddress(seed)
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			t.Fatalf("URLHashOf(%q): %v", address, err)
		}
		metadata = append(metadata, yacymodel.URLMetadata{Hash: hash, Address: address})
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

const recrawledPageURLSeed = "u1"

func postingOfRecrawledPage(word string, hits int) yacymodel.RWIPosting {
	entry := posting(word, recrawledPageURLSeed)
	entry.Hits = hits

	return entry
}

func (h harness) storedPosting(
	t *testing.T,
	entry yacymodel.RWIPosting,
) yacymodel.RWIPosting {
	t.Helper()

	var stored yacymodel.RWIPosting
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		found, _, err := h.index.PostingOf(tx, entry.WordHash, entry.URLHash)
		stored = found

		return err
	}); err != nil {
		t.Fatalf("PostingOf: %v", err)
	}

	return stored
}

func (h harness) receiveEveryPostingOfRecrawledPage(
	t *testing.T,
	postings []yacymodel.RWIPosting,
) {
	t.Helper()

	if _, err := h.pageReceiver.ReceiveEveryPostingOfPage(
		context.Background(),
		urlHash(recrawledPageURLSeed),
		postings,
	); err != nil {
		t.Fatalf("ReceiveEveryPostingOfPage: %v", err)
	}
}

func TestRecrawledPageLosesThePostingsOfTheWordsItNoLongerHas(t *testing.T) {
	h := openHarness(t, 0, 100)
	h.storeMetadata(t, recrawledPageURLSeed)
	departed, kept := postingOfRecrawledPage("w1", 1), postingOfRecrawledPage("w2", 3)
	h.receiveEveryPostingOfRecrawledPage(t, []yacymodel.RWIPosting{departed, kept})

	arrived := postingOfRecrawledPage("w3", 2)
	refreshed := postingOfRecrawledPage("w2", 7)
	h.receiveEveryPostingOfRecrawledPage(t, []yacymodel.RWIPosting{refreshed, arrived})

	if h.indexed(t, departed) {
		t.Error("a word the recrawled page no longer has kept its posting")
	}
	if !h.indexed(t, arrived) {
		t.Error("a word the recrawled page gained has no posting")
	}
	if got := h.storedPosting(t, refreshed).Hits; got != refreshed.Hits {
		t.Errorf("hits of the word the page kept = %d, want the recrawled %d",
			got, refreshed.Hits)
	}
}

func TestPeerPostingsLeaveEveryOtherPostingOfTheirURLInPlace(t *testing.T) {
	h := openHarness(t, 0, 100)
	h.storeMetadata(t, recrawledPageURLSeed)
	stored := []yacymodel.RWIPosting{
		postingOfRecrawledPage("w1", 1),
		postingOfRecrawledPage("w2", 1),
	}
	h.receiveEveryPostingOfRecrawledPage(t, stored)

	if _, err := h.receiver.Receive(
		context.Background(),
		[]yacymodel.RWIPosting{postingOfRecrawledPage("w3", 1)},
	); err != nil {
		t.Fatalf("Receive: %v", err)
	}

	for _, entry := range stored {
		if !h.indexed(t, entry) {
			t.Errorf("a partial batch from a peer dropped the posting of %q", entry.WordHash)
		}
	}
}

func TestRefusedPageLeavesThePostingsItWouldHaveReplaced(t *testing.T) {
	h := openHarnessAdmitting(t, 0, 100,
		func(admitter rwipostings.PostingAdmitter) rwipostings.PostingAdmitter {
			return refusedAdmitter{admitter: admitter, word: yacymodel.WordHash("w3")}
		})
	h.storeMetadata(t, recrawledPageURLSeed)
	departed, kept := postingOfRecrawledPage("w1", 1), postingOfRecrawledPage("w2", 3)
	h.receiveEveryPostingOfRecrawledPage(t, []yacymodel.RWIPosting{departed, kept})

	if _, err := h.pageReceiver.ReceiveEveryPostingOfPage(
		context.Background(),
		urlHash(recrawledPageURLSeed),
		[]yacymodel.RWIPosting{postingOfRecrawledPage("w2", 7), postingOfRecrawledPage("w3", 1)},
	); !errors.Is(err, errAdmissionRefused) {
		t.Fatalf("ReceiveEveryPostingOfPage error = %v, want the refused admission", err)
	}
	if !h.indexed(t, departed) {
		t.Error("a refused recrawl purged the posting of a word the page had before")
	}
	if got := h.storedPosting(t, kept).Hits; got != kept.Hits {
		t.Errorf("hits of the kept word = %d, want the %d stored before the refusal",
			got, kept.Hits)
	}
}
