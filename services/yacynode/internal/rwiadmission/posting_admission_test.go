package rwiadmission_test

import (
	"context"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpageversion"
)

const (
	busyPause  = 5 * time.Second
	postingCap = 3
)

type discardedHolds struct{}

func (discardedHolds) ObserveHeld(*vault.Txn, int)     {}
func (discardedHolds) ObserveReleased(*vault.Txn, int) {}

type recordedRefusals struct {
	postings map[rwiadmission.RefusalReason]int
}

func (r *recordedRefusals) ObserveRefused(reason rwiadmission.RefusalReason, postings int) {
	r.postings[reason] += postings
}

type discardedVerdicts struct{}

func (discardedVerdicts) ObserveVerdict(*vault.Txn, urlpageversion.Verdict) {}

var sendingPeer = rwiorigin.Of(yacymodel.WordHash("sending peer"), "203.0.113.7")

type stopwordThe struct{}

func (stopwordThe) Contains(word yacymodel.Hash, language yacymodel.Language) bool {
	return word == yacymodel.WordHash("the") && language == yacymodel.LanguageOfUndeclaredDocument
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

	index, references, admitter, purger, err := rwipostings.Open(
		context.Background(),
		v,
		yacymodel.DHTRingPartitions(1),
		nil,
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	versions, err := urlpageversion.Open(
		v,
		urlpageversion.Postings{
			Index:      index,
			References: references,
			Admitter:   admitter,
			Purger:     purger,
		},
		time.Now,
		urlpageversion.Config{
			SwapFloorPerHour: 100,
			SwapSharePerHour: 0.01,
			Observer:         discardedVerdicts{},
		},
	)
	if err != nil {
		t.Fatalf("urlpageversion.Open: %v", err)
	}
	escrow, err := rwiescrow.Open(
		context.Background(),
		v,
		versions,
		discardedHolds{},
		escrowCapacity,
		time.Now,
	)
	if err != nil {
		t.Fatalf("rwiescrow.Open: %v", err)
	}
	urlDirectory, _, _, urlReceiver, err := urlmeta.Open(
		context.Background(),
		v,
		nil,
		escrow,
		versions,
	)
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
			versions,
			escrow,
			stopwordThe{},
			rwiadmission.Config{PostingCap: postingCap, Pause: busyPause, Observer: refusals},
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

func (h harness) receive(t *testing.T, entries []yacymodel.RWIPosting) {
	t.Helper()

	if _, err := h.receiver.Receive(context.Background(), sendingPeer, entries); err != nil {
		t.Fatalf("Receive: %v", err)
	}
}

func TestKnownURLPostingJoinsIndex(t *testing.T) {
	h := openHarness(t, 0, 100)
	h.storeMetadata(t, "u1")
	entry := posting("w1", "u1")

	receipt, err := h.receiver.Receive(
		context.Background(),
		sendingPeer,
		[]yacymodel.RWIPosting{entry},
	)
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

func TestStopwordPostingNeitherJoinsTheIndexNorWaits(t *testing.T) {
	h := openHarness(t, 0, 100)
	h.storeMetadata(t, "u1")
	known := posting("the", "u1")
	unknown := posting("the", "u2")

	receipt, err := h.receiver.Receive(
		context.Background(),
		sendingPeer,
		[]yacymodel.RWIPosting{known, unknown},
	)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if h.indexed(t, known) {
		t.Fatal("stopword posting for a known url reached the index")
	}
	if got := h.heldCount(t); got != 0 {
		t.Fatalf("held count = %d, want the stopword posting dropped", got)
	}
	if len(receipt.UnknownURL) != 0 {
		t.Fatalf("UnknownURL = %v, want none: only a stopword names u2", receipt.UnknownURL)
	}
}

func TestUnknownURLPostingWaitsInEscrow(t *testing.T) {
	h := openHarness(t, 0, 100)
	entry := posting("w1", "u1")

	receipt, err := h.receiver.Receive(
		context.Background(),
		sendingPeer,
		[]yacymodel.RWIPosting{entry},
	)
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

	if _, err := h.receiver.Receive(context.Background(), sendingPeer,
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

	receipt, err := h.receiver.Receive(context.Background(), sendingPeer,
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
	h := openHarness(t, usedBytesOfAnEmptyNode(t)+1, 100)

	if _, err := h.receiver.Receive(context.Background(), sendingPeer,
		[]yacymodel.RWIPosting{posting("w1", "u1")},
	); err != nil {
		t.Fatalf("Receive: %v", err)
	}

	receipt, err := h.receiver.Receive(context.Background(), sendingPeer,
		[]yacymodel.RWIPosting{posting("w2", "u2")},
	)
	if err != nil {
		t.Fatalf("Receive over capacity: %v", err)
	}
	if !receipt.Busy || receipt.Pause != busyPause {
		t.Fatalf("receipt = %+v, want Busy with the configured pause", receipt)
	}
	if got := h.refusals.postings[rwiadmission.RefusalStorageFull]; got != 1 {
		t.Fatalf("postings refused for a full storage = %d, want all 1 received", got)
	}
}

func usedBytesOfAnEmptyNode(t *testing.T) int64 {
	t.Helper()

	usedBytes, err := openHarness(t, 0, 100).vault.UsedBytes(context.Background())
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	return usedBytes
}

func TestReceiveBusyWhenTheEscrowIsFull(t *testing.T) {
	h := openHarness(t, 0, 1)

	if _, err := h.receiver.Receive(context.Background(), sendingPeer,
		[]yacymodel.RWIPosting{posting("w1", "u1")},
	); err != nil {
		t.Fatalf("Receive: %v", err)
	}

	refused := []yacymodel.RWIPosting{posting("w2", "u2"), posting("w3", "u3")}
	receipt, err := h.receiver.Receive(context.Background(), sendingPeer, refused)
	if err != nil {
		t.Fatalf("Receive over the escrow capacity: %v", err)
	}
	if !receipt.Busy || receipt.Pause != busyPause {
		t.Fatalf("receipt = %+v, want Busy with the configured pause", receipt)
	}
	if got := h.heldCount(t); got != 1 {
		t.Fatalf("held count = %d, want the refused postings to leave nothing behind", got)
	}
	if got := h.refusals.postings[rwiadmission.RefusalEscrowFull]; got != len(refused) {
		t.Fatalf("postings refused for a full escrow = %d, want all %d received",
			got, len(refused))
	}
}

func TestReceiveHoldsAWholeTransferThatStartsBelowTheEscrowCapacity(t *testing.T) {
	h := openHarness(t, 0, 1)

	held := []yacymodel.RWIPosting{posting("w1", "u1"), posting("w2", "u2")}
	receipt, err := h.receiver.Receive(context.Background(), sendingPeer, held)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if receipt.Busy {
		t.Fatalf("receipt = %+v, want the transfer taken", receipt)
	}
	if got := h.heldCount(t); got != len(held) {
		t.Fatalf("held count = %d, want all %d received", got, len(held))
	}
}

func TestReceiveBusyWithMorePostingsThanTheCap(t *testing.T) {
	h := openHarness(t, 0, 100)
	h.storeMetadata(t, "u1")
	oversized := make([]yacymodel.RWIPosting, 0, postingCap+1)
	for seed := 0; seed <= postingCap; seed++ {
		oversized = append(oversized, posting("w"+strconv.Itoa(seed), "u1"))
	}

	receipt, err := h.receiver.Receive(context.Background(), sendingPeer, oversized)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if !receipt.Busy || receipt.Pause != busyPause {
		t.Fatalf("receipt = %+v, want Busy with the configured pause", receipt)
	}
	if h.indexed(t, oversized[0]) {
		t.Fatal("a posting of an oversized batch reached the index")
	}
	if got := h.refusals.postings[rwiadmission.RefusalTooManyPostings]; got != len(oversized) {
		t.Fatalf("postings refused as too many = %d, want %d", got, len(oversized))
	}
}

func TestReceiveReportsEachUnknownURLOnce(t *testing.T) {
	h := openHarness(t, 0, 100)

	receipt, err := h.receiver.Receive(context.Background(), sendingPeer, []yacymodel.RWIPosting{
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

func TestPeerPostingsLeaveEveryOtherPostingOfTheirURLInPlace(t *testing.T) {
	h := openHarness(t, 0, 100)
	h.storeMetadata(t, "u1")
	stored := []yacymodel.RWIPosting{posting("w1", "u1"), posting("w2", "u1")}
	h.receive(t, stored)

	h.receive(t, []yacymodel.RWIPosting{posting("w3", "u1")})

	for _, entry := range stored {
		if !h.indexed(t, entry) {
			t.Errorf("a later delivery dropped the posting of %q", entry.WordHash)
		}
	}
}

func (h harness) storeMetadataModifiedOn(t *testing.T, version yacymodel.MicroDate) {
	t.Helper()

	address := urlAddress("u1")
	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}
	metadata := yacymodel.URLMetadata{
		Hash:     hash,
		Address:  address,
		Modified: yacymodel.Some(yacymodel.CalendarDayOf(version.Time())),
	}
	if _, err := h.urls.Receive(
		context.Background(),
		[]yacymodel.URLMetadata{metadata},
	); err != nil {
		t.Fatalf("urls.Receive: %v", err)
	}
}

func postingOn(word string, version yacymodel.MicroDate) yacymodel.RWIPosting {
	entry := posting(word, "u1")
	entry.LastModified = version

	return entry
}

func TestNewerPostingWaitsForNewerMetadataThenReplacesTheOlderVersion(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := yacymodel.MicroDateFromTime(time.Now())
	older, newer := postingOn("w1", today-5), postingOn("w2", today)
	h.storeMetadataModifiedOn(t, today-5)
	h.receive(t, []yacymodel.RWIPosting{older})

	receipt, err := h.receiver.Receive(
		context.Background(),
		sendingPeer,
		[]yacymodel.RWIPosting{newer},
	)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if len(receipt.UnknownURL) != 1 || receipt.UnknownURL[0] != urlHash("u1") {
		t.Fatalf("UnknownURL = %v, want the url asked for its newer metadata", receipt.UnknownURL)
	}
	if !h.indexed(t, older) || h.indexed(t, newer) {
		t.Fatal("the url changed version before its newer metadata arrived")
	}

	h.storeMetadataModifiedOn(t, today)

	if h.indexed(t, older) || !h.indexed(t, newer) {
		t.Fatal("newer metadata did not replace the older version with the newer one")
	}
}

func TestOlderPostingLeavesTheIndexedVersionAlone(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := yacymodel.MicroDateFromTime(time.Now())
	indexed, older := postingOn("w1", today), postingOn("w2", today-5)
	h.storeMetadataModifiedOn(t, today)
	h.receive(t, []yacymodel.RWIPosting{indexed})

	h.receive(t, []yacymodel.RWIPosting{older})

	if !h.indexed(t, indexed) || h.indexed(t, older) {
		t.Fatal("an older posting changed the indexed version")
	}
	if got := h.heldCount(t); got != 0 {
		t.Fatalf("held count = %d, want the older posting dropped", got)
	}
}

func TestReceiveBusyWhenTheEscrowIsFullAndAPostingNeedsNewerMetadata(t *testing.T) {
	h := openHarness(t, 0, 1)
	today := yacymodel.MicroDateFromTime(time.Now())
	h.storeMetadataModifiedOn(t, today-5)
	h.receive(t, []yacymodel.RWIPosting{postingOn("w1", today-5), posting("w2", "u2")})

	newer := postingOn("w3", today)
	receipt, err := h.receiver.Receive(
		context.Background(),
		sendingPeer,
		[]yacymodel.RWIPosting{newer},
	)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if !receipt.Busy || h.heldCount(t) != 1 {
		t.Fatalf("receipt = %+v, held = %d, want busy and the newer posting left out",
			receipt, h.heldCount(t))
	}
}
