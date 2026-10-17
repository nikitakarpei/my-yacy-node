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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiindexedversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const (
	busyPause  = 5 * time.Second
	postingCap = 3
)

type discardedHolds struct{}

func (discardedHolds) ObserveHeld(*vault.Txn, int)     {}
func (discardedHolds) ObserveReleased(*vault.Txn, int) {}

type observedAdmission struct {
	outcomes map[rwiadmission.Outcome]int
}

func (o *observedAdmission) ObserveOutcome(outcome rwiadmission.Outcome, postings int) {
	o.outcomes[outcome] += postings
}

type admissionOpenedLater struct {
	admission rwiadmission.PostingAdmission
}

func (r *admissionOpenedLater) URLStored(
	tx *vault.Txn,
	url yacymodel.URLHash,
	version pagefingerprint.Fingerprint,
) error {
	return r.admission.URLStored(tx, url, version)
}

func (r *admissionOpenedLater) URLPurged(*vault.Txn, yacymodel.URLHash) error {
	return nil
}

var harnessStart = time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)

func harnessToday() yacymodel.MicroDate {
	return yacymodel.MicroDateFromTime(harnessStart)
}

var (
	sendingPeer   = rwiorigin.Of(yacymodel.WordHash("sending peer"), "203.0.113.7")
	neighbourPeer = rwiorigin.Of(yacymodel.WordHash("neighbour peer"), "203.0.113.8")
	otherPeer     = rwiorigin.Of(yacymodel.WordHash("other peer"), "198.51.100.9")
)

type stopwordThe struct{}

func (stopwordThe) Contains(word yacymodel.Hash, language yacymodel.Language) bool {
	return word == yacymodel.WordHash("the") && language == yacymodel.LanguageOfUndeclaredDocument
}

type harness struct {
	vault    *vault.Vault
	index    rwipostings.PostingIndex
	escrow   *rwiescrow.PostingEscrow
	urls     urlmeta.URLReceiver
	receiver rwiadmission.PostingAdmission
	observed *observedAdmission
	clock    *time.Time
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
	clock := harnessStart
	now := func() time.Time { return clock }

	index, references, admitter, purger, err := rwipostings.Open(
		context.Background(),
		v,
		yacymodel.DHTRingPartitions(1),
		nil,
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	indexedVersions, err := rwiindexedversion.Open(v)
	if err != nil {
		t.Fatalf("rwiindexedversion.Open: %v", err)
	}
	escrow, err := rwiescrow.Open(context.Background(), v, discardedHolds{}, escrowCapacity, now)
	if err != nil {
		t.Fatalf("rwiescrow.Open: %v", err)
	}
	relay := &admissionOpenedLater{}
	urlDirectory, _, _, urlReceiver, err := urlmeta.Open(
		context.Background(),
		v,
		now,
		nil,
		indexedVersions,
		relay,
	)
	if err != nil {
		t.Fatalf("urlmeta.Open: %v", err)
	}
	observed := &observedAdmission{outcomes: map[rwiadmission.Outcome]int{}}
	relay.admission = rwiadmission.Open(
		v,
		urlDirectory,
		escrow,
		indexedVersions,
		index,
		references,
		admitter,
		purger,
		stopwordThe{},
		observed,
		now,
		rwiadmission.Config{
			PostingCap:       postingCap,
			Pause:            busyPause,
			SwapFloorPerHour: 1,
		},
	)

	return harness{
		vault:    v,
		index:    index,
		escrow:   escrow,
		urls:     urlReceiver,
		receiver: relay.admission,
		observed: observed,
		clock:    &clock,
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
		TitleWords: yacymodel.TitleWordsOf(""),
		Hits:       1,
	}
}

func metadataOfPostingsTo(t *testing.T, seed string) yacymodel.URLMetadata {
	t.Helper()

	address := urlAddress(seed)
	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return yacymodel.URLMetadata{Hash: hash, Address: address, LocalLinks: 1}
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
		metadata = append(metadata, metadataOfPostingsTo(t, seed))
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
	if got := h.observed.outcomes[rwiadmission.OutcomeDroppedStopword]; got != 2 {
		t.Fatalf("postings dropped as stopwords = %d, want both", got)
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
	if got := h.observed.outcomes[rwiadmission.OutcomeRefusedStorageFull]; got != 1 {
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
	if got := h.observed.outcomes[rwiadmission.OutcomeRefusedEscrowFull]; got != len(refused) {
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
	if got := h.observed.outcomes[rwiadmission.OutcomeRefusedTooManyPostings]; got != len(
		oversized,
	) {
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

var _ rwiadmission.Escrow = (*rwiescrow.PostingEscrow)(nil)

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

	metadata := metadataOfPostingsTo(t, "u1")
	metadata.Modified = yacymodel.Some(yacymodel.CalendarDayOf(version.Time()))
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
	today := harnessToday()
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
	today := harnessToday()
	indexed, older := postingOn("w1", today), postingOn("w2", today-5)
	h.storeMetadataModifiedOn(t, today)
	h.receive(t, []yacymodel.RWIPosting{indexed})

	h.receive(t, []yacymodel.RWIPosting{older})

	if !h.indexed(t, indexed) || h.indexed(t, older) {
		t.Fatal("an older posting changed the indexed version")
	}
	if got := h.heldCount(
		t,
	); got != 0 ||
		h.observed.outcomes[rwiadmission.OutcomeDroppedOtherVersion] != 1 {
		t.Fatalf("held count = %d, other versions = %d, want the older posting dropped",
			got, h.observed.outcomes[rwiadmission.OutcomeDroppedOtherVersion])
	}
}

func TestPostingOfAnotherVersionOfTheSameDayIsDropped(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	h.storeMetadataModifiedOn(t, today)
	other := postingOn("w1", today)
	other.ExternalLinks = 7

	receipt, err := h.receiver.Receive(
		context.Background(),
		sendingPeer,
		[]yacymodel.RWIPosting{other},
	)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if h.indexed(t, other) || h.heldCount(t) != 0 || len(receipt.UnknownURL) != 0 {
		t.Fatalf("receipt = %+v, held = %d, want the posting dropped", receipt, h.heldCount(t))
	}
	if h.observed.outcomes[rwiadmission.OutcomeDroppedOtherVersion] != 1 {
		t.Fatalf(
			"other versions = %d, want 1",
			h.observed.outcomes[rwiadmission.OutcomeDroppedOtherVersion],
		)
	}
}

func TestReceiveBusyWhenTheEscrowIsFullAndAPostingNeedsNewerMetadata(t *testing.T) {
	h := openHarness(t, 0, 1)
	today := harnessToday()
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

func TestPostingDatedAfterTodayIsDropped(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	tomorrow, current := postingOn("w1", today+1), postingOn("w2", today)

	receipt, err := h.receiver.Receive(
		context.Background(),
		sendingPeer,
		[]yacymodel.RWIPosting{tomorrow, current},
	)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if h.heldCount(t) != 1 || len(receipt.UnknownURL) != 1 {
		t.Fatalf(
			"receipt = %+v, held = %d, want only today's posting waiting",
			receipt,
			h.heldCount(t),
		)
	}
	if h.observed.outcomes[rwiadmission.OutcomeDroppedFutureDate] != 1 {
		t.Fatalf("outcomes = %v, want one posting dated in the future", h.observed.outcomes)
	}
}

func TestPostingsOfTheIndexedVersionAddOnlyMissingWordsFromItsNetwork(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	h.storeMetadataModifiedOn(t, today)
	first := postingOn("w1", today)
	h.receive(t, []yacymodel.RWIPosting{first})
	resent := first
	resent.Hits = 7

	for _, c := range []struct {
		name    string
		origin  rwiorigin.Origin
		posting yacymodel.RWIPosting
		want    rwiadmission.Outcome
	}{
		{name: "missing word from the origin", origin: sendingPeer, posting: postingOn("w2", today), want: rwiadmission.OutcomeIndexedMissingWord},
		{name: "indexed word from the origin", origin: sendingPeer, posting: resent, want: rwiadmission.OutcomeDroppedSameVersion},
		{name: "missing word from another network", origin: otherPeer, posting: postingOn("w3", today), want: rwiadmission.OutcomeDroppedSameVersion},
		{name: "missing word from another peer of the network", origin: neighbourPeer, posting: postingOn("w4", today), want: rwiadmission.OutcomeIndexedMissingWord},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := h.observed.outcomes[c.want]
			if _, err := h.receiver.Receive(
				context.Background(),
				c.origin,
				[]yacymodel.RWIPosting{c.posting},
			); err != nil {
				t.Fatalf("Receive: %v", err)
			}
			if h.observed.outcomes[c.want] != before+1 {
				t.Fatalf("outcomes = %v, want one more %q", h.observed.outcomes, c.want)
			}
		})
	}
	if !h.indexed(t, postingOn("w2", today)) || h.indexed(t, postingOn("w3", today)) ||
		!h.indexed(t, postingOn("w4", today)) {
		t.Fatal("missing words admitted the wrong postings")
	}
	if h.indexedPosting(t, first).Hits != first.Hits {
		t.Fatal("a resent word replaced the indexed posting")
	}
}

func TestStoredMetadataOfAnotherVersionOfTheSameDaySwapsTheURL(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	old := postingOn("w1", today)
	h.storeMetadataModifiedOn(t, today)
	h.receive(t, []yacymodel.RWIPosting{old})

	metadata := metadataOfPostingsTo(t, "u1")
	metadata.Modified = yacymodel.Some(yacymodel.CalendarDayOf(today.Time()))
	metadata.LocalLinks = 9
	if _, err := h.urls.Receive(
		context.Background(),
		[]yacymodel.URLMetadata{metadata},
	); err != nil {
		t.Fatalf("urls.Receive: %v", err)
	}
	other := postingOn("w2", today)
	other.LocalLinks = 9
	h.receive(t, []yacymodel.RWIPosting{other})

	if h.indexed(t, old) || !h.indexed(t, other) {
		t.Fatal("postings of the stored version did not replace the indexed version")
	}
}

func TestSwapsPerNetworkStopAtTheHourlyLimit(t *testing.T) {
	h := openHarness(t, 0, 100)
	base := harnessToday() - 5
	h.storeMetadataModifiedOn(t, base)
	h.receive(t, []yacymodel.RWIPosting{postingOn("w1", base)})

	swapTo := func(origin rwiorigin.Origin, word string, day yacymodel.MicroDate) {
		t.Helper()
		h.storeMetadataModifiedOn(t, day)
		if _, err := h.receiver.Receive(
			context.Background(),
			origin,
			[]yacymodel.RWIPosting{postingOn(word, day)},
		); err != nil {
			t.Fatalf("Receive: %v", err)
		}
	}

	swapTo(sendingPeer, "w2", base+1)
	swapTo(sendingPeer, "w3", base+2)
	if h.observed.outcomes[rwiadmission.OutcomeIndexedNewerVersion] != 1 ||
		h.observed.outcomes[rwiadmission.OutcomeDroppedOverSwapLimit] != 1 {
		t.Fatalf("outcomes = %v, want one swap and one over the limit", h.observed.outcomes)
	}

	swapTo(otherPeer, "w3", base+2)
	*h.clock = h.clock.Add(time.Hour)
	swapTo(sendingPeer, "w4", base+3)

	if h.observed.outcomes[rwiadmission.OutcomeIndexedNewerVersion] != 3 {
		t.Fatalf(
			"outcomes = %v, want swaps from another network and an hour later",
			h.observed.outcomes,
		)
	}
}

func TestWaitingPostingOfAnotherVersionLeavesTheEscrowUnindexed(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	waiting := postingOn("w1", today)
	waiting.ExternalLinks = 7
	h.receive(t, []yacymodel.RWIPosting{waiting})

	h.storeMetadataModifiedOn(t, today)

	if h.heldCount(t) != 0 || h.indexed(t, waiting) {
		t.Fatalf(
			"held = %d, want the posting of another version released unindexed",
			h.heldCount(t),
		)
	}
	if h.observed.outcomes[rwiadmission.OutcomeDroppedOtherVersion] != 1 {
		t.Fatalf("outcomes = %v, want one other version", h.observed.outcomes)
	}
}

func TestWaitingPostingNewerThanTheStoredMetadataKeepsWaiting(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	waiting := postingOn("w1", today)
	h.receive(t, []yacymodel.RWIPosting{waiting})

	h.storeMetadataModifiedOn(t, today-1)

	if h.heldCount(t) != 1 || h.indexed(t, waiting) {
		t.Fatalf("held = %d, want the newer posting still waiting", h.heldCount(t))
	}
}

func (h harness) indexedPosting(t *testing.T, entry yacymodel.RWIPosting) yacymodel.RWIPosting {
	t.Helper()

	var indexed yacymodel.RWIPosting
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		stored, _, err := h.index.PostingOf(tx, entry.WordHash, entry.URLHash)
		indexed = stored

		return err
	}); err != nil {
		t.Fatalf("PostingOf: %v", err)
	}

	return indexed
}
