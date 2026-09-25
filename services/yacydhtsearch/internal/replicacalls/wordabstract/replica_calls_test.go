package wordabstract_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicacalls/wordabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheCoverageCountsTheDocumentsInTheAbstract(t *testing.T) {
	t.Parallel()

	coverage := wordabstract.New(peerCallsOfTheTests{}).CoverageFrom(
		peerasks.AnsweredWordAbstractAsk{Abstract: make([]yacymodel.URLHash, 3)},
	)

	if coverage != (replicaasks.Coverage{AmountOfDocumentsListed: 3, PeerSearched: true}) {
		t.Fatalf("the coverage = %+v, want the three documents of the abstract", coverage)
	}
}

func TestAnEmptyAbstractStillCoversThePartition(t *testing.T) {
	t.Parallel()

	coverage := wordabstract.New(peerCallsOfTheTests{}).CoverageFrom(
		peerasks.AnsweredWordAbstractAsk{},
	)

	if coverage != (replicaasks.Coverage{PeerSearched: true}) {
		t.Fatalf("the coverage = %+v, want a peer that searched and listed nothing", coverage)
	}
}

func TestTheReplicaOfAnAskIsItsPeerWordAndPartition(t *testing.T) {
	t.Parallel()

	ask := askOfTheTests()

	replica := wordabstract.New(peerCallsOfTheTests{}).ReplicaOf(ask)

	if replica != (replicaasks.Replica{Peer: ask.Peer, Word: ask.Word, Partition: ask.Partition}) {
		t.Fatalf("the replica = %+v, want the peer, word, and partition of %+v", replica, ask)
	}
}

func TestTheAnswerToAnAskIsTheAnswerOfItsPeer(t *testing.T) {
	t.Parallel()

	ask := askOfTheTests()

	answer, answered := wordabstract.New(peerCallsOfTheTests{answers: true}).AnswerTo(
		t.Context(), ask,
	)

	if !answered || answer.Ask.Peer != ask.Peer || len(answer.Abstract) != 1 {
		t.Fatalf("the answer = %+v, %t, want the answer of %s", answer, answered, ask.Peer.Address)
	}
}

func TestAPeerCallWithoutAnAnswerLeavesTheAskUnanswered(t *testing.T) {
	t.Parallel()

	_, answered := wordabstract.New(peerCallsOfTheTests{answers: false}).AnswerTo(
		t.Context(), askOfTheTests(),
	)

	if answered {
		t.Fatal("the ask was answered, want it unanswered")
	}
}

type peerCallsOfTheTests struct {
	answers bool
}

func (calls peerCallsOfTheTests) AskForWordAbstracts(
	_ context.Context,
	asks []peerasks.WordAbstractAsk,
) []peerasks.AnsweredWordAbstractAsk {
	if !calls.answers {
		return nil
	}

	return []peerasks.AnsweredWordAbstractAsk{
		{Ask: asks[0], Abstract: make([]yacymodel.URLHash, 1)},
	}
}

func askOfTheTests() peerasks.WordAbstractAsk {
	return peerasks.WordAbstractAsk{
		Peer: peerdirectory.AskablePeer{
			Hash:    yacymodel.WordHash("berlin-one"),
			Address: "berlin-one",
		},
		Partition: 3,
		Word:      yacymodel.WordHash("berlin"),
	}
}
