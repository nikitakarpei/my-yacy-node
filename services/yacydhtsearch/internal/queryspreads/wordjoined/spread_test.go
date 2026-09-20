package wordjoined_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	peerItemsCeiling               = 10
	urlMetadataAskDocumentsCeiling = 10
	crossCheckedDocumentsCeiling   = 10
	ringPartitionsExponent         = 0
	amountOfPeersHoldingOneWord    = 4
)

func ringPartitions(t *testing.T) yacymodel.DHTRingPartitions {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(ringPartitionsExponent)
	if err != nil {
		t.Fatalf("partitions from exponent: %v", err)
	}

	return partitions
}

func documentOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return hash
}

func metadataOf(t *testing.T, address string) yacymodel.URLMetadata {
	t.Helper()

	return yacymodel.URLMetadata{
		Hash:    documentOf(t, address),
		Address: address,
		Title:   "The title a peer sent",
	}
}

func peerNamed(spelledName string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{
		Hash:    yacymodel.WordHash(spelledName),
		Address: spelledName + ".example:8090",
	}
}

func chosenPeersOf(
	queryWords []yacymodel.Hash,
	peers ...peerdirectory.AskablePeer,
) peerchoice.ChosenPeersPerQueryWord {
	chosenPeers := make([]peerchoice.ChosenPeer, 0, len(peers))
	for _, peer := range peers {
		chosenPeers = append(chosenPeers, peerchoice.ChosenPeer{Peer: peer, Partition: 0})
	}

	chosenPeersPerQueryWord := make(peerchoice.ChosenPeersPerQueryWord, 0, len(queryWords))
	for _, queryWord := range queryWords {
		chosenPeersPerQueryWord = append(chosenPeersPerQueryWord, peerchoice.ChosenPeersOfQueryWord{
			QueryWord:   queryWord,
			ChosenPeers: chosenPeers,
		})
	}

	return chosenPeersPerQueryWord
}

type answeredAsksOfOneSpread struct {
	matchedAndHeldDocuments []peerasks.AnsweredMatchedAndHeldDocumentsAsk
	crossCheckedDocuments   []peerasks.AnsweredCrossCheckedDocumentsAsk
	urlMetadata             []peerasks.AnsweredURLMetadataAsk
	crossCheckedAsks        []peerasks.CrossCheckedDocumentsAsk
}

func (answered *answeredAsksOfOneSpread) AskForMatchedAndHeldDocuments(
	_ context.Context,
	_ []peerasks.MatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	return answered.matchedAndHeldDocuments
}

func (answered *answeredAsksOfOneSpread) AskForCrossCheckedDocuments(
	_ context.Context,
	asks []peerasks.CrossCheckedDocumentsAsk,
) []peerasks.AnsweredCrossCheckedDocumentsAsk {
	answered.crossCheckedAsks = asks

	return answered.crossCheckedDocuments
}

func (answered *answeredAsksOfOneSpread) AskForURLMetadata(
	_ context.Context,
	_ []peerasks.URLMetadataAsk,
) []peerasks.AnsweredURLMetadataAsk {
	return answered.urlMetadata
}

func spreadOver(
	t *testing.T,
	answered *answeredAsksOfOneSpread,
	asksForCrossCheckedDocuments bool,
) wordjoined.Spread {
	t.Helper()

	return wordjoined.New(
		answered,
		answered,
		urlMetadataAskDocumentsCeiling,
		asksForCrossCheckedDocuments,
		crossCheckedDocumentsCeiling,
		peerItemsCeiling,
		ringPartitions(t),
		amountOfPeersHoldingOneWord,
		wordjoined.WordJoinedSpreadObservers{},
	)
}

func answeredMatchedAndHeldDocumentsAsk(
	peer peerdirectory.AskablePeer,
	word yacymodel.Hash,
	matchedDocuments []peerasks.MatchedDocument,
	amountOfDocumentsHeld int,
) peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	documentsListedForTheWord := make([]yacymodel.URLHash, 0, len(matchedDocuments))
	for _, matchedDocument := range matchedDocuments {
		documentsListedForTheWord = append(
			documentsListedForTheWord, matchedDocument.Metadata.Hash,
		)
	}

	return peerasks.AnsweredMatchedAndHeldDocumentsAsk{
		Ask: peerasks.MatchedAndHeldDocumentsAsk{
			Peer: peer, Partition: 0, Word: word, ItemsCeiling: peerItemsCeiling,
		},
		DocumentsListedForTheWord:       documentsListedForTheWord,
		MatchedDocuments:                matchedDocuments,
		AmountOfDocumentsHeldForTheWord: yacymodel.Some(amountOfDocumentsHeld),
	}
}

func matchedDocumentOf(
	t *testing.T,
	address string,
	word yacymodel.Hash,
	hits int,
	textWords int,
) peerasks.MatchedDocument {
	t.Helper()

	return peerasks.MatchedDocument{
		Metadata: metadataOf(t, address),
		Posting: yacymodel.Some(yacymodel.RWIPosting{
			WordHash:  word,
			URLHash:   documentOf(t, address),
			Hits:      hits,
			TextWords: textWords,
		}),
	}
}

func foundDocumentOf(
	t *testing.T,
	answers queryanswers.AnsweredQuery,
	address string,
) queryanswers.FoundDocument {
	t.Helper()

	document := documentOf(t, address)
	for _, foundDocument := range answers.FoundDocuments {
		if foundDocument.Hash == document {
			return foundDocument
		}
	}
	t.Fatalf("the answers hold no document for %q", address)

	return queryanswers.FoundDocument{}
}

const (
	addressOfTheDocumentDealtToThePeer      = "https://dealt.example/"
	addressOfTheDocumentDealtToTheOtherPeer = "https://dealt-elsewhere.example/"
)

func TestACrossCheckAnswerCountsOnlyTheDocumentsTheAskDealtToThePeer(t *testing.T) {
	t.Parallel()

	query := searchquery.QueryFrom("berlin weather", "")
	queryWords := query.TermHashes()
	rarestWord, otherWord := queryWords[0], queryWords[1]
	claimingPeer, silentPeer := peerNamed("one"), peerNamed("two")
	dealtDocuments := []yacymodel.URLHash{
		documentOf(t, addressOfTheDocumentDealtToThePeer),
		documentOf(t, addressOfTheDocumentDealtToTheOtherPeer),
	}
	answered := &answeredAsksOfOneSpread{
		matchedAndHeldDocuments: []peerasks.AnsweredMatchedAndHeldDocumentsAsk{
			answeredMatchedAndHeldDocumentsAsk(
				claimingPeer,
				rarestWord,
				[]peerasks.MatchedDocument{
					matchedDocumentOf(t, addressOfTheDocumentDealtToThePeer, rarestWord, 3, 400),
					matchedDocumentOf(
						t, addressOfTheDocumentDealtToTheOtherPeer, rarestWord, 3, 400,
					),
				},
				2,
			),
			answeredMatchedAndHeldDocumentsAsk(claimingPeer, otherWord, nil, 900),
			answeredMatchedAndHeldDocumentsAsk(silentPeer, otherWord, nil, 900),
		},
	}
	answered.crossCheckedDocuments = []peerasks.AnsweredCrossCheckedDocumentsAsk{{
		Ask: peerasks.CrossCheckedDocumentsAsk{
			Peer: claimingPeer, Word: otherWord, Documents: dealtDocuments[:1],
		},
		DocumentsHeldForTheWord: dealtDocuments,
	}}

	answers := spreadOver(t, answered, true).
		SpreadOverPeers(t.Context(), query, chosenPeersOf(queryWords, claimingPeer, silentPeer))

	if len(answers.FoundDocuments) != 1 {
		t.Fatalf(
			"the answers hold %d documents, want only the one the ask dealt to the peer",
			len(answers.FoundDocuments),
		)
	}
	if answers.FoundDocuments[0].Hash != dealtDocuments[0] {
		t.Errorf(
			"the answers hold %s, want the document the ask dealt to the peer",
			answers.FoundDocuments[0].Address,
		)
	}
}

const addressOfTheDocumentEveryReplicaListed = "https://berlin.example/"

func answersOfThreeReplicasCounting(
	t *testing.T,
	hitsOfEachReplica []int,
	amountsOfWordsOfEachReplica []int,
) queryanswers.AnsweredQuery {
	t.Helper()

	query := searchquery.QueryFrom("berlin", "")
	queryWord := query.TermHashes()[0]
	peers := []peerdirectory.AskablePeer{
		peerNamed("one"), peerNamed("two"), peerNamed("three"),
	}
	answered := &answeredAsksOfOneSpread{}
	for replica, peer := range peers {
		answered.matchedAndHeldDocuments = append(
			answered.matchedAndHeldDocuments,
			answeredMatchedAndHeldDocumentsAsk(
				peer,
				queryWord,
				[]peerasks.MatchedDocument{matchedDocumentOf(
					t,
					addressOfTheDocumentEveryReplicaListed,
					queryWord,
					hitsOfEachReplica[replica],
					amountsOfWordsOfEachReplica[replica],
				)},
				1,
			),
		)
	}

	return spreadOver(t, answered, false).
		SpreadOverPeers(t.Context(), query, chosenPeersOf(query.TermHashes(), peers...))
}

func TestOneReplicaDoesNotOwnTheHitsTheAnswersCarry(t *testing.T) {
	t.Parallel()

	answers := answersOfThreeReplicasCounting(
		t, []int{900, 4, 6}, []int{400, 400, 400},
	)

	foundDocument := foundDocumentOf(t, answers, addressOfTheDocumentEveryReplicaListed)
	if hits := foundDocument.HitsPerQueryWord[yacymodel.WordHash("berlin")]; hits != 6 {
		t.Errorf("the document counts %d hits, want the median of what the replicas counted", hits)
	}
}

func TestOneReplicaDoesNotOwnTheAmountOfWordsTheAnswersCarry(t *testing.T) {
	t.Parallel()

	answers := answersOfThreeReplicasCounting(
		t, []int{4, 4, 4}, []int{90000, 400, 600},
	)

	foundDocument := foundDocumentOf(t, answers, addressOfTheDocumentEveryReplicaListed)
	if foundDocument.AmountOfWords != 600 {
		t.Errorf(
			"the document counts %d words, want the median of what the replicas counted",
			foundDocument.AmountOfWords,
		)
	}
}
