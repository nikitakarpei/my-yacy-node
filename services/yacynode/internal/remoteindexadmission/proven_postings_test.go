package remoteindexadmission_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

const crawlerRingExponent = 4

func crawledPageOf(day yacymodel.MicroDate, words ...string) yacymodel.PagePostings {
	postings := make([]yacymodel.RWIPosting, 0, len(words))
	for _, word := range words {
		postings = append(postings, postingOn(word, day))
	}

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(crawlerRingExponent)
	if err != nil {
		panic(err)
	}
	crawled, err := yacymodel.ProvenPagePostingsFrom(
		postings,
		yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		panic(err)
	}

	return crawled
}

func sentPagePostingsOf(
	crawled yacymodel.PagePostings,
	words ...string,
) yacymodel.PagePostings {
	sent := make([]yacymodel.RWIPosting, 0, len(words))
	for _, word := range words {
		for _, posting := range crawled.Postings() {
			if posting.WordHash == yacymodel.WordHash(word) {
				sent = append(sent, posting)
			}
		}
	}

	narrowed, _ := crawled.NarrowedTo(sent).Get()

	return narrowed
}

func rootedMetadataOf(
	t *testing.T,
	day yacymodel.MicroDate,
	crawled yacymodel.PagePostings,
) yacymodel.URLMetadata {
	t.Helper()

	metadata := metadataOn(t, "u1", day)
	metadata.PostingsRoot = crawled.Root()
	if !metadata.PostingsRoot.Present() {
		t.Fatal("the crawled page has no root")
	}

	return metadata
}

func (h harness) receiveProven(
	t *testing.T,
	origin rwiorigin.Origin,
	pagePostings yacymodel.PagePostings,
) remoteindexadmission.PostingReceipt {
	t.Helper()

	proofs := map[yacymodel.URLHash]yacymodel.PostingsProof{}
	if proof, proven := pagePostings.Proof().Get(); proven {
		proofs[pagePostings.URL()] = proof
	}
	receipt, err := h.receiver.Receive(
		context.Background(),
		origin,
		pagePostings.Postings(),
		proofs,
	)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}

	return receipt
}

func (h harness) receiveUnproven(
	t *testing.T,
	origin rwiorigin.Origin,
	postings []yacymodel.RWIPosting,
) {
	t.Helper()

	if _, err := h.receiver.Receive(context.Background(), origin, postings, nil); err != nil {
		t.Fatalf("Receive: %v", err)
	}
}

func (h harness) indexedPagePostingsOf(
	t *testing.T,
	postings ...yacymodel.RWIPosting,
) yacymodel.PagePostings {
	t.Helper()

	proof, proven := h.proofs.ProofsFor(context.Background(), postings)[postings[0].URLHash]
	if !proven {
		t.Fatalf("this node holds no proof for %d postings", len(postings))
	}

	held, err := yacymodel.ProvenPagePostingsFrom(postings, proof)
	if err != nil {
		t.Fatal(err)
	}

	return held
}

func TestPostingProvenUnderTheStoredRootJoinsFromAnyNetwork(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := crawledPageOf(today, "w1", "w2", "w3")
	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today, crawled))

	h.receiveProven(t, otherPeer, sentPagePostingsOf(crawled, "w1", "w2"))

	if !h.indexed(t, postingOn("w1", today)) || !h.indexed(t, postingOn("w2", today)) ||
		h.observed.received[remoteindexadmission.PostingIndexed] != 2 {
		t.Fatalf("outcomes = %v, want both proven postings indexed", h.observed.received)
	}
}

func TestUnprovenPostingOrOneUnderAnotherRootOfARootedVersionIsDropped(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := crawledPageOf(today, "w1", "w2")
	otherCrawl := crawledPageOf(today, "w1", "w3")
	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today, crawled))

	h.receiveUnproven(t, sendingPeer, crawled.Postings()[:1])
	h.receiveProven(t, sendingPeer, sentPagePostingsOf(otherCrawl, "w1"))

	if h.indexed(t, postingOn("w1", today)) ||
		h.observed.received[remoteindexadmission.PostingDroppedUnrootedVersion] != 1 ||
		h.observed.received[remoteindexadmission.PostingDroppedNotNewerVersion] != 1 {
		t.Fatalf("outcomes = %v, want one posting without a proof and one of another root",
			h.observed.received)
	}
}

func TestRootedVersionReplacesTheUnrootedOneOfItsDayWithASwapAndReleasesItsProvenPostings(
	t *testing.T,
) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	unrooted := postingOn("w9", today)
	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", today), metadataOn(t, "u2", today))
	h.receive(t, []yacymodel.RWIPosting{unrooted})
	crawled := crawledPageOf(today, "w1", "w2")

	receipt := h.receiveProven(t, otherPeer, sentPagePostingsOf(crawled, "w1"))
	if len(receipt.UnknownURL) != 1 || receipt.UnknownURL[0] != urlHash("u1") ||
		h.heldCount(t) != 1 {
		t.Fatalf("receipt = %+v, held = %d, want the proven posting waiting for its root",
			receipt, h.heldCount(t))
	}

	h.receiveMetadata(t, otherPeer, rootedMetadataOf(t, today, crawled))

	released := postingOn("w1", today)
	if h.indexed(t, unrooted) || !h.indexed(t, released) || h.heldCount(t) != 0 ||
		h.observed.urlMetadata[remoteindexadmission.URLMetadataReplacedUnrootedVersion] != 1 ||
		h.observed.released[remoteindexadmission.PostingIndexed] != 1 {
		t.Fatalf("metadata outcomes = %v, released = %v, want the rooted version in place",
			h.observed.urlMetadata, h.observed.released)
	}
	if got := h.indexedPagePostingsOf(t, released).Root(); got != crawled.Root() {
		t.Fatalf("root of the released posting = %+v, want %+v", got, crawled.Root())
	}
	rootedSecondPage := metadataOn(t, "u2", today)
	rootedSecondPage.PostingsRoot = crawled.Root()
	h.receiveMetadata(t, otherPeer, rootedSecondPage)
	if h.observed.urlMetadata[remoteindexadmission.URLMetadataDroppedTooManyReplacements] != 1 {
		t.Fatalf("outcomes = %v, want the replacement to have spent the swap",
			h.observed.urlMetadata)
	}
}

func TestSecondRootOfADayLeavesTheFirstInPlace(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	first, second := crawledPageOf(today, "w1", "w2"), crawledPageOf(today, "w1", "w3")
	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today, first))

	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today, second))
	h.receiveProven(t, otherPeer, sentPagePostingsOf(first, "w2"))

	if h.observed.urlMetadata[remoteindexadmission.URLMetadataDroppedNotNewerVersion] != 1 ||
		!h.indexed(t, postingOn("w2", today)) {
		t.Fatalf("metadata outcomes = %v, want the second root dropped and the first kept",
			h.observed.urlMetadata)
	}
}

func TestPostingsProvenOnArrivalAreIndexedWithAProofToOfferThemOn(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := crawledPageOf(today, "w1", "w2", "w3", "w4")
	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today, crawled))

	h.receiveProven(t, sendingPeer, sentPagePostingsOf(crawled, "w1", "w2"))
	h.receiveProven(t, otherPeer, sentPagePostingsOf(crawled, "w4"))

	offered := sentPagePostingsOf(crawled, "w1", "w4").Postings()
	if got := h.indexedPagePostingsOf(t, offered...).Root(); got != crawled.Root() {
		t.Fatalf("root of the postings offered on = %+v, want %+v", got, crawled.Root())
	}
}

func TestReleaseOfAProvenPostingMeetsTheRootItWaitedFor(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled, otherCrawl := crawledPageOf(today, "w1", "w2"), crawledPageOf(today, "w1", "w3")
	h.receiveProven(t, otherPeer, sentPagePostingsOf(otherCrawl, "w1"))
	h.receiveUnproven(t, otherPeer, crawled.Postings()[1:])

	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today, crawled))

	if h.indexed(t, postingOn("w1", today)) || h.indexed(t, postingOn("w2", today)) ||
		h.observed.released[remoteindexadmission.PostingDroppedNotNewerVersion] != 1 ||
		h.observed.released[remoteindexadmission.PostingDroppedUnrootedVersion] != 1 {
		t.Fatalf("released = %v, want the posting of another root and the unproven one dropped",
			h.observed.released)
	}
}

func TestPostingsProvenFromTwoNetworksJoinUnderTheRootTheyWaitedFor(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := crawledPageOf(today, "w1", "w2", "w3")
	h.receiveProven(t, sendingPeer, sentPagePostingsOf(crawled, "w1"))
	h.receiveProven(t, otherPeer, sentPagePostingsOf(crawled, "w2"))

	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today, crawled))

	released := sentPagePostingsOf(crawled, "w1", "w2").Postings()
	if !h.indexed(t, released[0]) || !h.indexed(t, released[1]) || h.heldCount(t) != 0 ||
		h.observed.released[remoteindexadmission.PostingIndexed] != 2 {
		t.Fatalf("released = %v, want both proven postings indexed", h.observed.released)
	}
	if got := h.indexedPagePostingsOf(t, released...).Root(); got != crawled.Root() {
		t.Fatalf("root of the released postings = %+v, want %+v", got, crawled.Root())
	}
}

func TestReleasedProvenPostingsOfOneURLAreWrittenAtOnce(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := crawledPageOf(today, "w1", "w2", "w3")
	h.receiveProven(t, otherPeer, sentPagePostingsOf(crawled, "w1", "w2"))
	h.written.writes = nil

	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today, crawled))

	if len(h.written.writes) == 0 {
		t.Fatal("no write of the released proven postings")
	}
	for _, write := range h.written.writes {
		if write.AmountOfPostings() != 2 {
			t.Errorf(
				"a write of released proven postings holds %d postings, want both",
				write.AmountOfPostings(),
			)
		}
	}
}

func TestNewerUnrootedMetadataLeavesTheRootedVersionAndItsProvenPostings(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := crawledPageOf(today-5, "w1", "w2")
	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today-5, crawled))
	h.receiveProven(t, sendingPeer, sentPagePostingsOf(crawled, "w1", "w2"))

	h.receiveMetadata(t, otherPeer, metadataOn(t, "u1", today))

	proven := crawled.Postings()
	if !h.indexed(t, proven[0]) || !h.indexed(t, proven[1]) ||
		h.observed.urlMetadata[remoteindexadmission.URLMetadataDroppedUnrootedVersion] != 1 {
		t.Fatalf("outcomes = %v, want the newer unrooted metadata dropped", h.observed.urlMetadata)
	}
	if got := h.indexedPagePostingsOf(t, proven...).Root(); got != crawled.Root() {
		t.Fatalf("root of the proven postings = %+v, want %+v", got, crawled.Root())
	}
}

func TestUnprovenNewerPostingForARootedVersionIsDroppedOnArrival(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := crawledPageOf(today-5, "w1")
	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today-5, crawled))

	receipt := h.receiveProven(t, sendingPeer, crawledPageOf(today, "w2"))
	newer := postingOn("w2", today)
	h.receiveUnproven(t, sendingPeer, []yacymodel.RWIPosting{newer})

	if len(receipt.UnknownURL) != 1 || h.heldCount(t) != 1 || h.indexed(t, newer) ||
		h.observed.received[remoteindexadmission.PostingDroppedUnrootedVersion] != 1 {
		t.Fatalf("outcomes = %v, held = %d, want only the newer proven posting held",
			h.observed.received, h.heldCount(t))
	}
}

func TestUnprovenNewerPostingForARootedVersionIsDroppedOnRelease(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	h.receive(t, []yacymodel.RWIPosting{postingOn("w1", today)})

	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today-1, crawledPageOf(today-1, "w1")))

	if h.heldCount(t) != 0 || h.indexed(t, postingOn("w1", today)) ||
		h.observed.released[remoteindexadmission.PostingDroppedUnrootedVersion] != 1 {
		t.Fatalf("released = %v, held = %d, want the unproven posting dropped",
			h.observed.released, h.heldCount(t))
	}
}

func TestUnrootedMetadataOfTheSameOrAnOlderDayForARootedVersionIsDroppedAsUnrooted(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := crawledPageOf(today-1, "w1")
	h.receiveMetadata(t, sendingPeer, rootedMetadataOf(t, today-1, crawled))

	h.receiveMetadata(t, otherPeer, metadataOn(t, "u1", today-1), metadataOn(t, "u1", today-2))

	if h.observed.urlMetadata[remoteindexadmission.URLMetadataDroppedUnrootedVersion] != 2 {
		t.Fatalf("outcomes = %v, want both unrooted records dropped as unrooted",
			h.observed.urlMetadata)
	}
}

func TestOlderRootedVersionReplacesANewerUnrootedOneAndReleasesItsProvenPostings(
	t *testing.T,
) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	crawled := crawledPageOf(today-5, "w1", "w2")
	h.receiveProven(t, otherPeer, sentPagePostingsOf(crawled, "w1", "w2"))

	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", today))
	if h.heldCount(t) != 2 || len(h.observed.released) != 0 {
		t.Fatalf("held = %d, released = %v, want both proven postings still waiting",
			h.heldCount(t), h.observed.released)
	}

	h.receiveMetadata(t, otherPeer, rootedMetadataOf(t, today-5, crawled))

	proven := crawled.Postings()
	if !h.indexed(t, proven[0]) || !h.indexed(t, proven[1]) || h.heldCount(t) != 0 ||
		h.observed.urlMetadata[remoteindexadmission.URLMetadataReplacedUnrootedVersion] != 1 ||
		h.observed.released[remoteindexadmission.PostingIndexed] != 2 {
		t.Fatalf("metadata outcomes = %v, released = %v, want the rooted version in place",
			h.observed.urlMetadata, h.observed.released)
	}
	if got := h.indexedPagePostingsOf(t, proven...).Root(); got != crawled.Root() {
		t.Fatalf("root of the released postings = %+v, want %+v", got, crawled.Root())
	}
}

func TestProvenOlderPostingForAnUnrootedVersionIsHeld(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", today))

	receipt := h.receiveProven(t, otherPeer, crawledPageOf(today-5, "w1"))

	if len(receipt.UnknownURL) != 1 || h.heldCount(t) != 1 ||
		h.observed.received[remoteindexadmission.PostingHeldForURLMetadata] != 1 {
		t.Fatalf("outcomes = %v, held = %d, want the proven older posting held",
			h.observed.received, h.heldCount(t))
	}
}
