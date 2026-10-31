package yacymodel_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestProvenPostingsJoinUnderTheRootOfTheirPage(t *testing.T) {
	postings := randomPagePostings(seededRandom(31), 5)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	provenPostings := whole.EachProvenPosting()

	joined, err := yacymodel.PagePostingsJoinedFrom(provenPostings[1:4])
	if err != nil {
		t.Fatalf("PagePostingsJoinedFrom: %v", err)
	}
	if joined.AmountOfPostings() != 3 || joined.Root() != whole.Root() {
		t.Fatalf("joined %d postings under %+v, want 3 under %+v",
			joined.AmountOfPostings(), joined.Root(), whole.Root())
	}
	if yacymodel.Some(provenPostings[0].Root()) != whole.Root() {
		t.Fatalf("root of a proven posting = %+v, want %+v", provenPostings[0].Root(), whole.Root())
	}
	if _, err := yacymodel.PagePostingsJoinedFrom(nil); !errors.Is(
		err,
		yacymodel.ErrBadPagePostings,
	) {
		t.Errorf("PagePostingsJoinedFrom(nil) = %v, want ErrBadPagePostings", err)
	}
}

func TestPagePostingsJoinedFromRefusesProvenPostingsOfDifferentRoots(t *testing.T) {
	postings := randomPagePostings(seededRandom(36), 3)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	underOtherPostings := wholePagePostingsOf(t, postings[:2], partitionsFrom(t, 4))
	underOtherPartitions := wholePagePostingsOf(t, postings, partitionsFrom(t, 5))

	for name, joining := range map[string][]yacymodel.ProvenPosting{
		"other postings":   {whole.EachProvenPosting()[0], underOtherPostings.EachProvenPosting()[1]},
		"other partitions": {whole.EachProvenPosting()[0], underOtherPartitions.EachProvenPosting()[1]},
	} {
		if _, err := yacymodel.PagePostingsJoinedFrom(joining); !errors.Is(
			err,
			yacymodel.ErrBadPagePostings,
		) {
			t.Errorf("%s: PagePostingsJoinedFrom = %v, want ErrBadPagePostings", name, err)
		}
	}
}

func TestJoinedProvenPostingsTakeTheMinimalProofOfTheirPostings(t *testing.T) {
	random := seededRandom(34)
	for _, exponent := range partitionExponents {
		postings := randomPagePostings(random, 2+random.IntN(200))
		whole := wholePagePostingsOf(t, postings, partitionsFrom(t, exponent))
		provenPostings := whole.EachProvenPosting()
		kept := provenPostings[:1+random.IntN(len(provenPostings)-1)]

		joined, err := yacymodel.PagePostingsJoinedFrom(kept)
		if err != nil {
			t.Fatalf("PagePostingsJoinedFrom: %v", err)
		}
		minimal := narrowedTo(t, whole, joined.Postings())
		if !bytes.Equal(proofOf(t, joined).Bytes(), proofOf(t, minimal).Bytes()) {
			t.Fatalf("joined proof = %x, want the minimal %x",
				proofOf(t, joined).Bytes(), proofOf(t, minimal).Bytes())
		}
	}
}

func TestProvenPostingFromRefusesAProofBesideNoPathOfItsPosting(t *testing.T) {
	postings := randomPagePostings(seededRandom(33), 2)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	provenPosting := whole.EachProvenPosting()[0]

	if _, err := yacymodel.ProvenPostingFrom(postings[1], provenPosting.Proof()); err == nil {
		t.Error("a proof beside another posting's path was accepted")
	}
	rebuilt, err := yacymodel.ProvenPostingFrom(provenPosting.Posting(), provenPosting.Proof())
	if err != nil || yacymodel.Some(rebuilt.Root()) != whole.Root() {
		t.Fatalf("rebuilt proven posting = %+v, %v, want it under %+v", rebuilt, err, whole.Root())
	}
}

func TestProvenPostingFromRefusesAPostingWithoutItsHashes(t *testing.T) {
	posting := randomPagePostings(seededRandom(35), 1)[0]
	proof := yacymodel.EmptyPostingsProof(partitionsFrom(t, 4))
	withoutWord, withoutURL := posting, posting
	withoutWord.WordHash = yacymodel.Hash{}
	withoutURL.URLHash = yacymodel.URLHash{}

	for name, malformed := range map[string]yacymodel.RWIPosting{
		"no word hash": withoutWord,
		"no url hash":  withoutURL,
	} {
		if _, err := yacymodel.ProvenPostingFrom(malformed, proof); !errors.Is(
			err,
			yacymodel.ErrBadPagePostings,
		) {
			t.Errorf("%s: ProvenPostingFrom = %v, want ErrBadPagePostings", name, err)
		}
	}
}
