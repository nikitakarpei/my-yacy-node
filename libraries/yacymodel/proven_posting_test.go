package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestProvenPostingsJoinUnderTheRootOfTheirPage(t *testing.T) {
	postings := randomPagePostings(seededRandom(31), 5)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	provenPostings := whole.EachProvenPosting()

	joined, present := yacymodel.ProvenPagePostingsJoinedFrom(provenPostings[1:4]).Get()
	if !present {
		t.Fatal("proven postings of one page do not join")
	}
	if joined.AmountOfPostings() != 3 || joined.Root() != whole.Root() {
		t.Fatalf("joined %d postings under %+v, want 3 under %+v",
			joined.AmountOfPostings(), joined.Root(), whole.Root())
	}
	if yacymodel.Some(provenPostings[0].Root()) != whole.Root() {
		t.Fatalf("root of a proven posting = %+v, want %+v", provenPostings[0].Root(), whole.Root())
	}
	if yacymodel.ProvenPagePostingsJoinedFrom(nil).Present() {
		t.Error("no proven postings join into page postings")
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
