package yacymodel_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPagePostingsJoinUnderTheRootOfTheirPage(t *testing.T) {
	postings := randomPagePostings(seededRandom(31), 5)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	pagePostings := whole.EachPagePosting()

	joined, err := yacymodel.PagePostingsJoinedFrom(pagePostings[1:4])
	if err != nil {
		t.Fatalf("PagePostingsJoinedFrom: %v", err)
	}
	if joined.AmountOfPostings() != 3 || joined.Root() != whole.Root() {
		t.Fatalf("joined %d postings under %+v, want 3 under %+v",
			joined.AmountOfPostings(), joined.Root(), whole.Root())
	}
	if _, err := yacymodel.PagePostingsJoinedFrom(nil); !errors.Is(
		err,
		yacymodel.ErrBadPagePostings,
	) {
		t.Errorf("PagePostingsJoinedFrom(nil) = %v, want ErrBadPagePostings", err)
	}
}

func TestUnprovenPagePostingsJoinWithoutProofOrRoot(t *testing.T) {
	postings := randomPagePostings(seededRandom(37), 3)
	unproven, err := yacymodel.UnprovenPagePostingsFrom(postings)
	if err != nil {
		t.Fatal(err)
	}

	joined, err := yacymodel.PagePostingsJoinedFrom(unproven.EachPagePosting()[:2])
	if err != nil {
		t.Fatalf("PagePostingsJoinedFrom: %v", err)
	}
	if joined.AmountOfPostings() != 2 || joined.Proof().Present() || joined.Root().Present() {
		t.Fatalf("joined %d postings with proof %t and root %t, want 2 without either",
			joined.AmountOfPostings(), joined.Proof().Present(), joined.Root().Present())
	}
}

func TestPagePostingsJoinedFromRefusesPagePostingsOfDifferentRoots(t *testing.T) {
	postings := randomPagePostings(seededRandom(36), 3)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	underOtherPostings := wholePagePostingsOf(t, postings[:2], partitionsFrom(t, 4))
	underOtherPartitions := wholePagePostingsOf(t, postings, partitionsFrom(t, 5))
	unproven, err := yacymodel.UnprovenPagePostingsFrom(postings)
	if err != nil {
		t.Fatal(err)
	}

	for name, joining := range map[string][]yacymodel.PagePosting{
		"other postings":   {whole.EachPagePosting()[0], underOtherPostings.EachPagePosting()[1]},
		"other partitions": {whole.EachPagePosting()[0], underOtherPartitions.EachPagePosting()[1]},
		"proven, then unproven": {
			whole.EachPagePosting()[0],
			unproven.EachPagePosting()[1],
		},
		"unproven, then proven": {
			unproven.EachPagePosting()[0],
			whole.EachPagePosting()[1],
		},
	} {
		if _, err := yacymodel.PagePostingsJoinedFrom(joining); !errors.Is(
			err,
			yacymodel.ErrBadPagePostings,
		) {
			t.Errorf("%s: PagePostingsJoinedFrom = %v, want ErrBadPagePostings", name, err)
		}
	}
}

func TestJoinedPagePostingsTakeTheMinimalProofOfTheirPostings(t *testing.T) {
	random := seededRandom(34)
	for _, exponent := range partitionExponents {
		postings := randomPagePostings(random, 2+random.IntN(200))
		whole := wholePagePostingsOf(t, postings, partitionsFrom(t, exponent))
		pagePostings := whole.EachPagePosting()
		kept := pagePostings[:1+random.IntN(len(pagePostings)-1)]

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

func TestProvenPagePostingFromRefusesAProofBesideNoPathOfItsPosting(t *testing.T) {
	postings := randomPagePostings(seededRandom(33), 2)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	pagePosting := whole.EachPagePosting()[0]
	proof := pagePostingProofOf(t, pagePosting)

	if _, err := yacymodel.ProvenPagePostingFrom(postings[1], proof); err == nil {
		t.Error("a proof beside another posting's path was accepted")
	}
	rebuilt, err := yacymodel.ProvenPagePostingFrom(pagePosting.Posting(), proof)
	if err != nil || rebuilt.Root() != whole.Root() {
		t.Fatalf("rebuilt page posting = %+v, %v, want it under %+v", rebuilt, err, whole.Root())
	}
}

func TestPagePostingFromRefusesAPostingWithoutItsHashes(t *testing.T) {
	posting := randomPagePostings(seededRandom(35), 1)[0]
	proof := yacymodel.EmptyPostingsProof(partitionsFrom(t, 4))
	withoutWord, withoutURL := posting, posting
	withoutWord.WordHash = yacymodel.Hash{}
	withoutURL.URLHash = yacymodel.URLHash{}

	for name, malformed := range map[string]yacymodel.RWIPosting{
		"no word hash": withoutWord,
		"no url hash":  withoutURL,
	} {
		if _, err := yacymodel.ProvenPagePostingFrom(malformed, proof); !errors.Is(
			err,
			yacymodel.ErrBadPagePostings,
		) {
			t.Errorf("%s: ProvenPagePostingFrom = %v, want ErrBadPagePostings", name, err)
		}
		if _, err := yacymodel.UnprovenPagePostingFrom(malformed); !errors.Is(
			err,
			yacymodel.ErrBadPagePostings,
		) {
			t.Errorf("%s: UnprovenPagePostingFrom = %v, want ErrBadPagePostings", name, err)
		}
	}
	unproven, err := yacymodel.UnprovenPagePostingFrom(posting)
	if err != nil || unproven.Posting() != posting || unproven.Proof().Present() ||
		unproven.Root().Present() {
		t.Fatalf("unproven page posting = %+v, %v, want the posting alone", unproven, err)
	}
}
