package remoteindexadmission_test

import (
	"context"
	"maps"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpostingpurge"
)

const (
	busyPause  = 5 * time.Second
	postingCap = 3
)

type discardedPurges struct{}

func (discardedPurges) ObservePostingsPurgedWithURL(*vault.Txn, yacymodel.URLHash, int) {}

type observedAdmission struct {
	received    map[remoteindexadmission.ReceivedPostingOutcome]int
	released    map[remoteindexadmission.PostingPlacement]int
	urlMetadata map[remoteindexadmission.URLMetadataOutcome]int
}

func (o *observedAdmission) ObserveReceivedPostings(
	outcome remoteindexadmission.ReceivedPostingOutcome,
	postings int,
) {
	o.received[outcome] += postings
}

func (o *observedAdmission) ObserveReleasedPostings(
	placement remoteindexadmission.PostingPlacement,
	postings int,
) {
	o.released[placement] += postings
}

func (o *observedAdmission) receivedAndPlaced(placement remoteindexadmission.PostingPlacement) int {
	return o.received[remoteindexadmission.ReceivedPostingOutcome(placement)]
}

func (o *observedAdmission) ObserveURLMetadata(
	outcome remoteindexadmission.URLMetadataOutcome,
	records int,
) {
	o.urlMetadata[outcome] += records
}

var harnessStart = time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)

func harnessToday() yacymodel.MicroDate {
	return yacymodel.MicroDateFromTime(harnessStart)
}

var (
	sendingPeer   = rwiorigin.Of("203.0.113.7")
	neighbourPeer = rwiorigin.Of("203.0.113.8")
	otherPeer     = rwiorigin.Of("198.51.100.9")
)

type stopwordThe struct{}

func (stopwordThe) Contains(word yacymodel.Hash, language yacymodel.Language) bool {
	return word == yacymodel.WordHash("the") && language == yacymodel.LanguageOfUndeclaredDocument
}

type harness struct {
	vault    *vault.Vault
	index    rwipostings.PostingIndex
	escrow   *rwiescrow.PostingEscrow
	urls     remoteindexadmission.URLMetadataIntake
	crawled  urlmeta.URLMetadataWriter
	receiver remoteindexadmission.PostingAdmission
	observed *observedAdmission
	clock    *time.Time
}

func openHarness(t *testing.T, quotaBytes int64, escrowCapacity int) harness {
	t.Helper()

	v, err := vault.New(
		vaultenginetest.EngineRepeatingWrites(memoryvault.OpenEngine(quotaBytes)),
		nil,
		nil,
		wallclock.Clock{},
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

	index, writer, purger, err := rwipostings.Open(
		context.Background(),
		v,
		yacymodel.DHTRingPartitions{},
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	escrow, err := rwiescrow.Open(context.Background(), v, escrowCapacity, now)
	if err != nil {
		t.Fatalf("rwiescrow.Open: %v", err)
	}
	urlDirectory, urlEvictor, urlWriter, err := urlmeta.Open(
		context.Background(),
		v,
		nil,
		urlpostingpurge.New(purger, discardedPurges{}),
	)
	if err != nil {
		t.Fatalf("urlmeta.Open: %v", err)
	}
	observed := &observedAdmission{
		received:    map[remoteindexadmission.ReceivedPostingOutcome]int{},
		released:    map[remoteindexadmission.PostingPlacement]int{},
		urlMetadata: map[remoteindexadmission.URLMetadataOutcome]int{},
	}
	postingAdmission, urlMetadataIntake, err := remoteindexadmission.Open(
		v,
		urlDirectory,
		urlEvictor,
		urlWriter,
		escrow,
		index,
		writer,
		stopwordThe{},
		observed,
		now,
		remoteindexadmission.Config{
			PostingCap: postingCap,
			Pause:      busyPause,
			SwapLimit:  remoteindexadmission.SwapLimit{FloorPerHour: 1},
		},
	)
	if err != nil {
		t.Fatalf("remoteindexadmission.Open: %v", err)
	}

	return harness{
		vault:    v,
		index:    index,
		escrow:   escrow,
		urls:     urlMetadataIntake,
		crawled:  urlWriter,
		receiver: postingAdmission,
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
	if _, err := h.urls.Receive(context.Background(), sendingPeer, metadata); err != nil {
		t.Fatalf("urls.Receive: %v", err)
	}
}

func (h harness) receive(t *testing.T, entries []yacymodel.RWIPosting) {
	t.Helper()

	if _, err := h.receiver.Receive(context.Background(), sendingPeer, entries); err != nil {
		t.Fatalf("Receive: %v", err)
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
	if got := h.observed.received[remoteindexadmission.PostingDroppedStopword]; got != 2 {
		t.Fatalf("postings dropped as stopwords = %d, want both", got)
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
	wantReceived := map[remoteindexadmission.ReceivedPostingOutcome]int{
		remoteindexadmission.PostingHeldForURLMetadata: 1,
	}
	wantReleased := map[remoteindexadmission.PostingPlacement]int{
		remoteindexadmission.PostingIndexed: 1,
	}
	if !maps.Equal(h.observed.received, wantReceived) ||
		!maps.Equal(h.observed.released, wantReleased) {
		t.Fatalf(
			"received = %v, released = %v, want the posting held on receipt and indexed on release",
			h.observed.received,
			h.observed.released,
		)
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
	if h.indexed(t, unknown) || h.heldCount(t) != 1 {
		t.Fatalf("held = %d, want only the posting for the unknown url waiting", h.heldCount(t))
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
	if got := h.observed.received[remoteindexadmission.PostingRefusedStorageFull]; got != 1 {
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
	if got := h.observed.received[remoteindexadmission.PostingRefusedEscrowFull]; got != len(
		refused,
	) {
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
	if got := h.observed.received[remoteindexadmission.PostingRefusedTooManyPostings]; got != len(
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

var _ remoteindexadmission.Escrow = (*rwiescrow.PostingEscrow)(nil)

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
		context.Background(), sendingPeer,
		[]yacymodel.URLMetadata{metadata},
	); err != nil {
		t.Fatalf("urls.Receive: %v", err)
	}
}

func (h harness) writeCrawledMetadata(t *testing.T, metadata yacymodel.URLMetadata) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := h.crawled.Write(tx, metadata, urlmeta.SourceThisNode())

		return err
	}); err != nil {
		t.Fatalf("Write: %v", err)
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
		h.observed.receivedAndPlaced(remoteindexadmission.PostingDroppedNotNewerVersion) != 1 {
		t.Fatalf("held count = %d, other versions = %d, want the older posting dropped",
			got, h.observed.receivedAndPlaced(remoteindexadmission.PostingDroppedNotNewerVersion))
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
	if h.observed.receivedAndPlaced(remoteindexadmission.PostingDroppedNotNewerVersion) != 1 {
		t.Fatalf(
			"other versions = %d, want 1",
			h.observed.receivedAndPlaced(remoteindexadmission.PostingDroppedNotNewerVersion),
		)
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
	if h.observed.received[remoteindexadmission.PostingDroppedFutureDate] != 1 {
		t.Fatalf("outcomes = %v, want one posting dated in the future", h.observed.received)
	}
}

func TestMissingWordsOfTheCurrentVersionComeOnlyFromTheNetworkThatSentItsMetadata(t *testing.T) {
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
		want    remoteindexadmission.PostingPlacement
	}{
		{name: "missing word from the network", origin: sendingPeer, posting: postingOn("w2", today), want: remoteindexadmission.PostingIndexed},
		{name: "indexed word from the network", origin: sendingPeer, posting: resent, want: remoteindexadmission.PostingDroppedAlreadyIndexed},
		{name: "missing word from another network", origin: otherPeer, posting: postingOn("w3", today), want: remoteindexadmission.PostingDroppedPageFromOtherSource},
		{name: "missing word from another peer of the network", origin: neighbourPeer, posting: postingOn("w4", today), want: remoteindexadmission.PostingIndexed},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := h.observed.receivedAndPlaced(c.want)
			if _, err := h.receiver.Receive(
				context.Background(),
				c.origin,
				[]yacymodel.RWIPosting{c.posting},
			); err != nil {
				t.Fatalf("Receive: %v", err)
			}
			if h.observed.receivedAndPlaced(c.want) != before+1 {
				t.Fatalf("outcomes = %v, want one more %q", h.observed.received, c.want)
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

func TestPeerPostingsAddNoWordToAPageThisNodeCrawled(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := metadataOfPostingsTo(t, "u1")
	crawled.Modified = yacymodel.Some(yacymodel.CalendarDayOf(today.Time()))
	h.writeCrawledMetadata(t, crawled)

	sent := postingOn("w1", today)
	h.receive(t, []yacymodel.RWIPosting{sent})

	if h.indexed(t, sent) ||
		h.observed.receivedAndPlaced(remoteindexadmission.PostingDroppedPageFromOtherSource) != 1 {
		t.Fatalf("outcomes = %v, want the peer posting dropped", h.observed.received)
	}
}

func TestPeerMetadataOfAnotherVersionOfTheSameDayLeavesTheIndexedVersion(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	indexed := postingOn("w1", today)
	h.storeMetadataModifiedOn(t, today)
	h.receive(t, []yacymodel.RWIPosting{indexed})

	sameDay := metadataOn(t, "u1", today)
	sameDay.LocalLinks = 9
	other := postingOn("w2", today)
	other.LocalLinks = 9
	h.receiveMetadata(t, sendingPeer, sameDay)
	h.receive(t, []yacymodel.RWIPosting{other})

	if !h.indexed(t, indexed) || h.indexed(t, other) {
		t.Fatal("peer metadata of the same day replaced the indexed version")
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
	if h.observed.released[remoteindexadmission.PostingDroppedNotNewerVersion] != 1 {
		t.Fatalf("outcomes = %v, want one other version", h.observed.released)
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
