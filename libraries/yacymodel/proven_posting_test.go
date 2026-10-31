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
