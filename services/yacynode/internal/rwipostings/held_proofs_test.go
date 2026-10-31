package rwipostings_test

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func pagePostingsFrom(t *testing.T, words ...string) yacymodel.PagePostings {
	t.Helper()
	postings := make([]yacymodel.RWIPosting, 0, len(words))
	for _, word := range words {
		postings = append(postings, posting(word, "page"))
	}

	return wholePagePostingsOf(t, postings)
}

func wholePagePostingsOf(t *testing.T, postings []yacymodel.RWIPosting) yacymodel.PagePostings {
	t.Helper()
	pagePostings, err := yacymodel.ProvenPagePostingsFrom(
		postings,
		yacymodel.EmptyPostingsProof(ringPartitions(t)),
	)
	if err != nil {
		t.Fatal(err)
	}

	return pagePostings
}

func narrowedTo(
	t *testing.T,
	pagePostings yacymodel.PagePostings,
	keptPostings []yacymodel.RWIPosting,
) yacymodel.PagePostings {
	t.Helper()
	narrowed, present := pagePostings.NarrowedTo(keptPostings).Get()
	if !present {
		t.Fatalf("no narrowing to %d postings", len(keptPostings))
	}

	return narrowed
}

func TestAHandoffKeepsTheRemainingPostingsOfAURLLeadingToItsRoot(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2", "w3", "w4")
	h.write(t, crawled)

	h.handOff(t, crawled.Postings()[1])
	h.handOff(t, crawled.Postings()[3])

	remainingPostings := []yacymodel.RWIPosting{crawled.Postings()[0], crawled.Postings()[2]}
	if got := h.rootFromProofsFor(t, remainingPostings); got != crawled.Root() {
		t.Fatalf("root of the remaining postings = %+v, want %+v", got, crawled.Root())
	}
}

func TestPostingsWrittenWithTheirProofsMergeIntoTheHeldProof(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2", "w3", "w4")
	h.write(t, narrowedTo(t, crawled, crawled.Postings()[:1]))

	h.write(t, narrowedTo(t, crawled, crawled.Postings()[2:3]))

	heldPostings := []yacymodel.RWIPosting{crawled.Postings()[0], crawled.Postings()[2]}
	if got := h.rootFromProofsFor(t, heldPostings); got != crawled.Root() {
		t.Fatalf("root of the held postings = %+v, want %+v", got, crawled.Root())
	}
}

func TestPurgingTheLastPostingOfAURLDeletesItsHeldProof(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2")
	h.write(t, crawled)

	h.update(t, func(tx *vault.Txn) error {
		_, err := h.purger.PurgePosting(tx, crawled.Postings()[0].WordHash, urlHash("page"))

		return err
	})
	if got := h.rootFromProofsFor(t, crawled.Postings()[1:]); got != crawled.Root() {
		t.Fatalf("root after purging one of two postings = %+v, want %+v", got, crawled.Root())
	}
	h.handOff(t, crawled.Postings()[1])

	if proofs := h.proofs.ProofsFor(t.Context(), crawled.Postings()); len(proofs) > 0 {
		t.Fatalf("a url without postings kept a held proof: %v", proofs)
	}
}

func TestPurgingAURLDeletesItsHeldProof(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2")
	h.write(t, crawled)

	h.purgePostingsOf(t, urlHash("page"))

	if proofs := h.proofs.ProofsFor(t.Context(), crawled.Postings()); len(proofs) > 0 {
		t.Fatalf("a purged url kept its held proof: %v", proofs)
	}
}

func TestAWriteWithoutAProofLeavesTheHeldProofsAlone(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2")
	h.write(t, crawled)
	elsewhere := posting("w1", "other")

	h.writeEach(t, crawled.Postings()[0], elsewhere)

	if got := h.rootFromProofsFor(t, crawled.Postings()); got != crawled.Root() {
		t.Fatalf("root after a write without a proof = %+v, want %+v", got, crawled.Root())
	}
	if proofs := h.proofs.ProofsFor(
		t.Context(),
		[]yacymodel.RWIPosting{elsewhere},
	); len(proofs) > 0 {
		t.Fatalf("a write without a proof made a held proof: %v", proofs)
	}
}

func TestARecrawlAfterAHandoffHoldsOnlyTheProofOfTheNewPage(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2", "w3", "w4")
	h.write(t, crawled)
	h.handOff(t, crawled.Postings()[1])
	recrawledPostings := pagePostingsFrom(t, "w1", "w5").Postings()
	recrawledPostings[0].Hits = 5
	recrawled := wholePagePostingsOf(t, recrawledPostings)

	h.update(t, func(tx *vault.Txn) error {
		if _, err := h.purger.PurgePostingsOf(tx, urlHash("page")); err != nil {
			return err
		}

		return h.writer.Write(tx, recrawled)
	})

	proof := h.proofs.ProofsFor(t.Context(), recrawled.Postings())[urlHash("page")]
	minimalProof, _ := narrowedTo(t, recrawled, recrawled.Postings()).Proof().Get()
	if !bytes.Equal(proof.Bytes(), minimalProof.Bytes()) {
		t.Fatalf("proof of the recrawled page = %x, want %x", proof.Bytes(), minimalProof.Bytes())
	}
	if got := h.rootFromProofsFor(t, recrawled.Postings()); got != recrawled.Root() {
		t.Fatalf("root of the recrawled page = %+v, want %+v", got, recrawled.Root())
	}
}

func TestProofsForGivesNoProofForPostingsTheRecrawlReplaced(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2")
	h.write(t, crawled)
	recrawledPostings := pagePostingsFrom(t, "w1", "w2").Postings()
	recrawledPostings[0].Hits = 5
	recrawled := wholePagePostingsOf(t, recrawledPostings)

	h.update(t, func(tx *vault.Txn) error {
		if _, err := h.purger.PurgePostingsOf(tx, urlHash("page")); err != nil {
			return err
		}

		return h.writer.Write(tx, recrawled)
	})

	if proofs := h.proofs.ProofsFor(t.Context(), crawled.Postings()); len(proofs) > 0 {
		t.Fatalf("ProofsFor proved postings the recrawl replaced: %v", proofs)
	}
}

func TestProofsForGivesNoProofForAPostingReplacedInPlace(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2")
	h.write(t, crawled)
	replaced := crawled.Postings()[0]
	replaced.Hits = 5

	h.writeEach(t, replaced)

	if proofs := h.proofs.ProofsFor(t.Context(), crawled.Postings()); len(proofs) > 0 {
		t.Fatalf("ProofsFor proved a posting replaced in place: %v", proofs)
	}
}

func TestAHandoffOfAWordPartitionNarrowsOnlyTheProvenURL(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2")
	h.write(t, crawled)
	unproven := posting("w1", urlSeedInPartitionOf(t, urlHash("page")))
	h.writeEach(t, unproven)

	h.handOff(t, crawled.Postings()[0])

	if _, found := h.postingOf(t, unproven.WordHash, unproven.URLHash); found {
		t.Fatal("the handoff kept the posting of the unproven url")
	}
	if got := h.rootFromProofsFor(t, crawled.Postings()[1:]); got != crawled.Root() {
		t.Fatalf("root of the remaining posting = %+v, want %+v", got, crawled.Root())
	}
	if proofs := h.proofs.ProofsFor(
		t.Context(),
		[]yacymodel.RWIPosting{unproven},
	); len(proofs) > 0 {
		t.Fatalf("the unproven url got a held proof: %v", proofs)
	}
}

func urlSeedInPartitionOf(t *testing.T, url yacymodel.URLHash) string {
	t.Helper()
	partitions := ringPartitions(t)
	for number := 0; ; number++ {
		seed := "unproven" + strconv.Itoa(number)
		if partitions.PartitionOf(urlHash(seed)) == partitions.PartitionOf(url) {
			return seed
		}
	}
}

func TestProofsForGivesNoProofsWhenTheReadFails(t *testing.T) {
	h := openHarness(t)
	crawled := pagePostingsFrom(t, "w1", "w2")
	h.write(t, crawled)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if proofs := h.proofs.ProofsFor(ctx, crawled.Postings()); len(proofs) > 0 {
		t.Fatalf("ProofsFor gave proofs after a failed read: %v", proofs)
	}
}
