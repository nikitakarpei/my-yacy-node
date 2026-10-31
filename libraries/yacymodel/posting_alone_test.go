package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPostingsAloneJoinUnderTheRootOfTheirPage(t *testing.T) {
	postings := randomPagePostings(seededRandom(31), 5)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	eachAlone := whole.EachPostingAlone()

	joined, present := yacymodel.PagePostingsJoinedFrom(eachAlone[1:4]).Get()
	if !present {
		t.Fatal("postings alone of one page do not join")
	}
	if joined.AmountOfPostings() != 3 || joined.Root() != whole.Root() {
		t.Fatalf("joined %d postings under %+v, want 3 under %+v",
			joined.AmountOfPostings(), joined.Root(), whole.Root())
	}
	if eachAlone[0].Root() != whole.Root() {
		t.Fatalf("root of a posting alone = %+v, want %+v", eachAlone[0].Root(), whole.Root())
	}
}

func TestPostingsAloneJoinWithoutProofWhenOneHasNone(t *testing.T) {
	postings := randomPagePostings(seededRandom(32), 2)
	proven := wholePagePostingsOf(t, postings, partitionsFrom(t, 4)).EachPostingAlone()[0]
	unproven := yacymodel.UnprovenPostingAloneFrom(postings[1])

	joined, present := yacymodel.PagePostingsJoinedFrom(
		[]yacymodel.PostingAlone{proven, unproven},
	).Get()
	if !present || joined.Proof().Present() {
		t.Fatalf("joined = %+v, want both postings without a proof", joined)
	}
	if yacymodel.PagePostingsJoinedFrom(nil).Present() {
		t.Error("no postings alone join into page postings")
	}
}

func TestPostingAloneFromRefusesAProofBesideNoPathOfItsPosting(t *testing.T) {
	postings := randomPagePostings(seededRandom(33), 2)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	alone := whole.EachPostingAlone()[0]

	if _, err := yacymodel.PostingAloneFrom(postings[1], alone.Proof()); err == nil {
		t.Error("a proof beside another posting's path was accepted")
	}
	rebuilt, err := yacymodel.PostingAloneFrom(alone.Posting(), alone.Proof())
	if err != nil || rebuilt.Root() != whole.Root() {
		t.Fatalf("rebuilt posting alone = %+v, %v, want it under %+v", rebuilt, err, whole.Root())
	}
}
