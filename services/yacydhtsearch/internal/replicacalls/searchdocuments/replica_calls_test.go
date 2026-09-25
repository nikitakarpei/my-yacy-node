package searchdocuments_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicacalls/searchdocuments"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheCoverageCountsTheAbstractFirstThenTheMatchedDocumentsThenTheAmountHeld(
	t *testing.T,
) {
	t.Parallel()

	answers := map[string]struct {
		answer                  peerasks.AnsweredSearchDocumentsAsk
		amountOfDocumentsListed int
	}{
		"abstract": {
			answer: peerasks.AnsweredSearchDocumentsAsk{
				Abstract:                        make([]yacymodel.URLHash, 3),
				MatchedDocuments:                make([]peerasks.MatchedDocument, 2),
				AmountOfDocumentsHeldForTheWord: yacymodel.Some(7),
			},
			amountOfDocumentsListed: 3,
		},
		"matched documents": {
			answer: peerasks.AnsweredSearchDocumentsAsk{
				MatchedDocuments:                make([]peerasks.MatchedDocument, 2),
				AmountOfDocumentsHeldForTheWord: yacymodel.Some(7),
			},
			amountOfDocumentsListed: 2,
		},
		"amount held": {
			answer: peerasks.AnsweredSearchDocumentsAsk{
				AmountOfDocumentsHeldForTheWord: yacymodel.Some(7),
			},
			amountOfDocumentsListed: 7,
		},
		"negative amount held": {
			answer: peerasks.AnsweredSearchDocumentsAsk{
				AmountOfDocumentsHeldForTheWord: yacymodel.Some(-1),
			},
			amountOfDocumentsListed: 0,
		},
		"nothing counted": {
			answer:                  peerasks.AnsweredSearchDocumentsAsk{},
			amountOfDocumentsListed: 0,
		},
	}
	for name, answer := range answers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			coverage := searchdocuments.New(peerCallsOfTheTests{}).CoverageFrom(answer.answer)

			if coverage.AmountOfDocumentsListed != answer.amountOfDocumentsListed {
				t.Fatalf(
					"the answer listed %d documents, want %d",
					coverage.AmountOfDocumentsListed, answer.amountOfDocumentsListed,
				)
			}
		})
	}
}

func TestTheCoverageTellsThatThePeerSearched(t *testing.T) {
	t.Parallel()

	coverage := searchdocuments.New(peerCallsOfTheTests{}).CoverageFrom(
		peerasks.AnsweredSearchDocumentsAsk{PeerSearched: true},
	)

	if coverage != (replicaasks.Coverage{PeerSearched: true}) {
		t.Fatalf("the coverage = %+v, want a peer that searched and listed nothing", coverage)
	}
}

func TestTheReplicaOfAnAskIsItsPeerWordAndPartition(t *testing.T) {
	t.Parallel()

	ask := askOfTheTests()

	replica := searchdocuments.New(peerCallsOfTheTests{}).ReplicaOf(ask)

	if replica != (replicaasks.Replica{Peer: ask.Peer, Word: ask.Word, Partition: ask.Partition}) {
		t.Fatalf("the replica = %+v, want the peer, word, and partition of %+v", replica, ask)
	}
}

func TestTheAnswerToAnAskIsTheAnswerOfItsPeer(t *testing.T) {
	t.Parallel()

	ask := askOfTheTests()

	answer, answered := searchdocuments.New(peerCallsOfTheTests{answers: true}).AnswerTo(
		t.Context(), ask,
	)

	if !answered || answer.Ask.Peer != ask.Peer || !answer.PeerSearched {
		t.Fatalf("the answer = %+v, %t, want the answer of %s", answer, answered, ask.Peer.Address)
	}
}

func TestAPeerCallWithoutAnAnswerLeavesTheAskUnanswered(t *testing.T) {
	t.Parallel()

	_, answered := searchdocuments.New(peerCallsOfTheTests{answers: false}).AnswerTo(
		t.Context(), askOfTheTests(),
	)

	if answered {
		t.Fatal("the ask was answered, want it unanswered")
	}
}

type peerCallsOfTheTests struct {
	answers bool
}

func (calls peerCallsOfTheTests) AskForSearchDocuments(
	_ context.Context,
	asks []peerasks.SearchDocumentsAsk,
) []peerasks.AnsweredSearchDocumentsAsk {
	if !calls.answers {
		return nil
	}

	return []peerasks.AnsweredSearchDocumentsAsk{{Ask: asks[0], PeerSearched: true}}
}

func askOfTheTests() peerasks.SearchDocumentsAsk {
	return peerasks.SearchDocumentsAsk{
		Peer: peerdirectory.AskablePeer{
			Hash:    yacymodel.WordHash("berlin-one"),
			Address: "berlin-one",
		},
		Partition: 3,
		Word:      yacymodel.WordHash("berlin"),
	}
}
