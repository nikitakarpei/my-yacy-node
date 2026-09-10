package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedWordJoinedSearch struct {
	AmountOfQueryWords                     int
	AmountOfQueryWordsNoPeerHeld           int
	AmountOfPeersAskedForHeldDocuments     int
	AmountOfPeersThatNamedHeldDocuments    int
	AmountOfPeersHoldingAQueryWord         int
	AmountOfJoinedDocuments                int
	AmountOfJoinedDocumentsAlreadyReported int
	AmountOfReportedItems                  int
	AmountOfReportedItemsWithAPosting      int
	DocumentsEachPeerHoldsForAQueryWord    []int
	AmountOfDocumentsToAskMetadataFor      int
	AmountOfDocumentsThatCameBack          int
	TimeSpent                              time.Duration
}

//nolint:revive // argument-limit: the stages one word joined search passes
func performedWordJoinedSearchFrom(
	queryWords []yacymodel.Hash,
	heldDocumentsAsks []peerasks.HeldDocumentsAsk,
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
	joinedDocuments map[yacymodel.URLHash]struct{},
	reportedItems [][]searchresult.Item,
	documentsToAskMetadataFor map[yacymodel.URLHash]struct{},
	answeredURLMetadataAsks []peerasks.AnsweredURLMetadataAsk,
	timeSpent time.Duration,
) PerformedWordJoinedSearch {
	asksThatHeldADocument := answeredAsksThatHeldADocument(answeredHeldDocumentsAsks)

	return PerformedWordJoinedSearch{
		AmountOfQueryWords: len(queryWords),
		AmountOfQueryWordsNoPeerHeld: len(queryWords) -
			amountOfWordsAcrossAnsweredAsks(asksThatHeldADocument),
		AmountOfPeersAskedForHeldDocuments: amountOfPeersAcrossAsks(heldDocumentsAsks),
		AmountOfPeersThatNamedHeldDocuments: amountOfPeersAcrossAnsweredAsks(
			answeredHeldDocumentsAsks,
		),
		AmountOfPeersHoldingAQueryWord: amountOfPeersAcrossAnsweredAsks(
			asksThatHeldADocument,
		),
		AmountOfJoinedDocuments:                len(joinedDocuments),
		AmountOfJoinedDocumentsAlreadyReported: amountOfDocumentsAmong(reportedItems),
		AmountOfReportedItems:                  amountOfReportedItems(answeredHeldDocumentsAsks),
		AmountOfReportedItemsWithAPosting: amountOfReportedItemsWithAPosting(
			answeredHeldDocumentsAsks,
		),
		DocumentsEachPeerHoldsForAQueryWord: documentsEachPeerHoldsForAQueryWord(
			answeredHeldDocumentsAsks,
		),
		AmountOfDocumentsToAskMetadataFor: len(documentsToAskMetadataFor),
		AmountOfDocumentsThatCameBack: amountOfDocumentsThatCameBack(
			documentsToAskMetadataFor,
			answeredURLMetadataAsks,
		),
		TimeSpent: timeSpent,
	}
}

func amountOfDocumentsAmong(itemsOfEachAnsweredAsk [][]searchresult.Item) int {
	documents := map[yacymodel.URLHash]struct{}{}
	for _, items := range itemsOfEachAnsweredAsk {
		for _, item := range items {
			documents[item.Hash] = struct{}{}
		}
	}

	return len(documents)
}

func amountOfReportedItems(answeredAsks []peerasks.AnsweredHeldDocumentsAsk) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		amount += len(answeredAsk.Items)
	}

	return amount
}

func amountOfReportedItemsWithAPosting(answeredAsks []peerasks.AnsweredHeldDocumentsAsk) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		amount += answeredAsk.AmountOfItemsWithAPosting
	}

	return amount
}

func documentsEachPeerHoldsForAQueryWord(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) []int {
	documentsHeld := make([]int, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		documentsHeld = append(documentsHeld, answeredAsk.AmountOfDocumentsHeldForTheWord)
	}

	return documentsHeld
}

func answeredAsksThatHeldADocument(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) []peerasks.AnsweredHeldDocumentsAsk {
	kept := make([]peerasks.AnsweredHeldDocumentsAsk, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.Documents) == 0 {
			continue
		}
		kept = append(kept, answeredAsk)
	}

	return kept
}

func amountOfWordsAcrossAnsweredAsks(answeredAsks []peerasks.AnsweredHeldDocumentsAsk) int {
	words := map[yacymodel.Hash]struct{}{}
	for _, answeredAsk := range answeredAsks {
		words[answeredAsk.Ask.Word] = struct{}{}
	}

	return len(words)
}

func amountOfPeersAcrossAsks(asks []peerasks.HeldDocumentsAsk) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, ask := range asks {
		peers[ask.Peer] = struct{}{}
	}

	return len(peers)
}

func amountOfPeersAcrossAnsweredAsks(answeredAsks []peerasks.AnsweredHeldDocumentsAsk) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, answeredAsk := range answeredAsks {
		peers[answeredAsk.Ask.Peer] = struct{}{}
	}

	return len(peers)
}

func amountOfDocumentsThatCameBack(
	documentsToAskMetadataFor map[yacymodel.URLHash]struct{},
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) int {
	documentsThatCameBack := map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range answeredAsks {
		for _, item := range answeredAsk.Items {
			if _, asked := documentsToAskMetadataFor[item.Hash]; !asked {
				continue
			}
			documentsThatCameBack[item.Hash] = struct{}{}
		}
	}

	return len(documentsThatCameBack)
}
