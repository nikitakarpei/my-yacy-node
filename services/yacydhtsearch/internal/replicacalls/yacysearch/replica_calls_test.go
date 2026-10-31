package yacysearch_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicacalls/yacysearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
)

const matchedDocumentsCeiling = 10

func TestTheAskToTheReplicaCarriesTheWordPartitionAskAndTheWants(t *testing.T) {
	t.Parallel()

	peers := &peersOfTheTests{}
	ask := wordpartitionasks.Ask{
		Word:             yacymodel.WordHash("berlin"),
		Partition:        3,
		ReplicasInOrder:  []peerdirectory.AskablePeer{peerAt("first"), peerAt("second")},
		ExcludedWords:    []yacymodel.Hash{yacymodel.WordHash("rain")},
		Language:         languageFrom(t, "de"),
		DocumentsToMatch: []yacymodel.URLHash{urlhashtest.URLHashOf(t, "https://a.example/")},
	}

	replicaCallsOf(peers).Put(t.Context(), ask, peerAt("second"))

	if len(peers.asks) != 1 {
		t.Fatalf("the replica calls put %d asks, want one", len(peers.asks))
	}
	put := peers.asks[0]
	if put.Peer != peerAt("second") || put.Word != ask.Word ||
		!slices.Equal(put.ExcludedWords, ask.ExcludedWords) || put.Language != ask.Language ||
		!slices.Equal(put.DocumentsToMatch, ask.DocumentsToMatch) || !put.Abstract ||
		put.MatchedDocumentsCeiling != yacymodel.Some(matchedDocumentsCeiling) {
		t.Fatalf(
			"the replica calls put %+v, want the ask to the second replica with the wants",
			put,
		)
	}
}

func TestTheAnswerListsEachDocumentOfTheAbstractAndTheMatchedDocumentsOnce(t *testing.T) {
	t.Parallel()

	inBoth := urlhashtest.URLHashOf(t, "https://both.example/")
	onlyListed := urlhashtest.URLHashOf(t, "https://listed.example/")
	onlyMatched := urlhashtest.URLHashOf(t, "https://matched.example/")
	peers := &peersOfTheTests{answer: peerasks.AnsweredSearchDocumentsAsk{
		Abstract: yacymodel.URLHashesFrom(onlyListed, inBoth),
		MatchedDocuments: []peerasks.MatchedDocument{
			{Metadata: yacymodel.URLMetadata{Hash: onlyMatched}},
			{Metadata: yacymodel.URLMetadata{Hash: inBoth}},
		},
	}}

	answer, _ := replicaCallsOf(peers).Put(t.Context(), askForBerlin(), peerAt("first"))

	if want := yacymodel.URLHashesFrom(onlyListed, inBoth, onlyMatched); !reflect.DeepEqual(
		answer.ListedDocuments, want,
	) {
		t.Fatalf("the replica listed %v, want %v",
			answer.ListedDocuments.InHashOrder(), want.InHashOrder())
	}
}

func TestADocumentMatchedTwiceKeepsItsLastMatch(t *testing.T) {
	t.Parallel()

	document := urlhashtest.URLHashOf(t, "https://document.example/")
	other := urlhashtest.URLHashOf(t, "https://other.example/")
	posting := yacymodel.RWIPosting{Hits: 3}
	peers := &peersOfTheTests{answer: peerasks.AnsweredSearchDocumentsAsk{
		MatchedDocuments: []peerasks.MatchedDocument{
			{Metadata: yacymodel.URLMetadata{Hash: document}},
			{Metadata: yacymodel.URLMetadata{Hash: other}},
			{Metadata: yacymodel.URLMetadata{Hash: document}, Posting: yacymodel.Some(posting)},
		},
	}}

	answer, _ := replicaCallsOf(peers).Put(t.Context(), askForBerlin(), peerAt("first"))

	want := []wordpartitionasks.MatchedDocument{
		{Metadata: yacymodel.URLMetadata{Hash: document}, Posting: yacymodel.Some(posting)},
		{Metadata: yacymodel.URLMetadata{Hash: other}},
	}
	if !reflect.DeepEqual(answer.MatchedDocuments, want) {
		t.Fatalf("the replica matched %+v, want %+v", answer.MatchedDocuments, want)
	}
}

func TestAnAnswerThatOnlyMatchesDocumentsListsThem(t *testing.T) {
	t.Parallel()

	peers := &peersOfTheTests{answer: peerasks.AnsweredSearchDocumentsAsk{
		MatchedDocuments: []peerasks.MatchedDocument{
			{Metadata: yacymodel.URLMetadata{Hash: urlhashtest.URLHashOf(t, "https://a.example/")}},
			{Metadata: yacymodel.URLMetadata{Hash: urlhashtest.URLHashOf(t, "https://b.example/")}},
		},
	}}

	answer, _ := replicaCallsOf(peers).Put(t.Context(), askForBerlin(), peerAt("first"))

	if answer.ListedDocuments.Amount() != 2 {
		t.Fatalf("the replica listed %+v, want both matched documents", answer.ListedDocuments)
	}
}

func TestTheAnswerTellsTheReplicaTheAmountHeldAndTheSearch(t *testing.T) {
	t.Parallel()

	peers := &peersOfTheTests{answer: peerasks.AnsweredSearchDocumentsAsk{
		AmountOfDocumentsHeldForTheWord: yacymodel.Some(5),
		PeerSearched:                    true,
	}}

	answer, answered := replicaCallsOf(peers).Put(t.Context(), askForBerlin(), peerAt("first"))

	if !answered || answer.Holder != peerAt("first") ||
		answer.AmountOfDocumentsHeld != yacymodel.Some(5) || !answer.Searched ||
		answer.ListedDocuments.Amount() != 0 {
		t.Fatalf(
			"the replica answered %+v %t, want the first replica holding five documents after "+
				"a search",
			answer,
			answered,
		)
	}
}

func TestAReplicaThatGivesNoAnswerIsNotAnswered(t *testing.T) {
	t.Parallel()

	peers := &peersOfTheTests{silent: true}

	_, answered := replicaCallsOf(peers).Put(t.Context(), askForBerlin(), peerAt("first"))

	if answered {
		t.Fatal("the replica calls answered, want no answer from a silent replica")
	}
}

type peersOfTheTests struct {
	answer peerasks.AnsweredSearchDocumentsAsk
	silent bool
	asks   []peerasks.SearchDocumentsAsk
}

func (peers *peersOfTheTests) AskForSearchDocuments(
	_ context.Context,
	asks []peerasks.SearchDocumentsAsk,
) []peerasks.AnsweredSearchDocumentsAsk {
	peers.asks = append(peers.asks, asks...)
	if peers.silent {
		return nil
	}
	answer := peers.answer
	answer.Ask = asks[0]

	return []peerasks.AnsweredSearchDocumentsAsk{answer}
}

func replicaCallsOf(peers *peersOfTheTests) yacysearch.ReplicaCalls {
	return yacysearch.New(peers, yacysearch.Wants{
		Abstract:                true,
		MatchedDocumentsCeiling: yacymodel.Some(matchedDocumentsCeiling),
	})
}

func askForBerlin() wordpartitionasks.Ask {
	return wordpartitionasks.Ask{
		Word:            yacymodel.WordHash("berlin"),
		ReplicasInOrder: []peerdirectory.AskablePeer{peerAt("first")},
	}
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash(address), Address: address}
}

func languageFrom(t *testing.T, code string) yacymodel.Language {
	t.Helper()

	language, err := yacymodel.ParseLanguage(code)
	if err != nil {
		t.Fatalf("ParseLanguage(%s): %v", code, err)
	}

	return language
}
