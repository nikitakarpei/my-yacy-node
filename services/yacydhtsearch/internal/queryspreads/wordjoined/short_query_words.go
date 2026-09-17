package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type shortQueryWord struct {
	word  yacymodel.Hash
	peers []peerdirectory.AskablePeer
}

func shortQueryWordsOf(
	queryWords []yacymodel.Hash,
	peersPerQueryWord []peerchoice.PeersOfQueryWord,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	amountOfDocumentsHeldPerQueryWord map[yacymodel.Hash]int,
	anchorWord yacymodel.Hash,
) []shortQueryWord {
	completeQueryWords := completeQueryWordsAmong(queryWords, peersPerQueryWord, answeredAsks)
	shortQueryWords := make([]shortQueryWord, 0, len(queryWords))
	for index, queryWord := range queryWords {
		if queryWord == anchorWord || slices.Contains(completeQueryWords, queryWord) {
			continue
		}
		shortQueryWords = append(shortQueryWords, shortQueryWord{
			word:  queryWord,
			peers: peersPerQueryWord[index].Peers(),
		})
	}
	slices.SortStableFunc(shortQueryWords, fewestDocumentsFirst(amountOfDocumentsHeldPerQueryWord))

	return shortQueryWords
}

func completeQueryWordsAmong(
	queryWords []yacymodel.Hash,
	peersPerQueryWord []peerchoice.PeersOfQueryWord,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []yacymodel.Hash {
	completeQueryWords := make([]yacymodel.Hash, 0, len(queryWords))
	for index, queryWord := range queryWords {
		if partitionsWithACompleteAnswerOf(
			peersPerQueryWord[index], answeredAsks, queryWord,
		) < int(peersPerQueryWord[index].Partitions) {
			continue
		}
		completeQueryWords = append(completeQueryWords, queryWord)
	}

	return completeQueryWords
}

func partitionsWithACompleteAnswerOf(
	peersOfQueryWord peerchoice.PeersOfQueryWord,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	queryWord yacymodel.Hash,
) int {
	partitionOfEachPeer := partitionOfEachPeerOf(peersOfQueryWord)
	partitionsWithACompleteAnswer := map[uint]struct{}{}
	for _, answeredAsk := range answeredAsks {
		if answeredAsk.Ask.Word != queryWord || !answerIsComplete(answeredAsk) {
			continue
		}
		partition, chosen := partitionOfEachPeer[answeredAsk.Ask.Peer.Hash]
		if !chosen {
			continue
		}
		partitionsWithACompleteAnswer[partition] = struct{}{}
	}

	return len(partitionsWithACompleteAnswer)
}

func answerIsComplete(answeredAsk peerasks.AnsweredMatchedAndHeldDocumentsAsk) bool {
	amountOfDocumentsHeld, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()

	return counted && amountOfDocumentsHeld <= len(answeredAsk.DocumentsHeldForTheWord)
}

func fewestDocumentsFirst(
	amountOfDocumentsHeldPerQueryWord map[yacymodel.Hash]int,
) func(first, second shortQueryWord) int {
	return func(first, second shortQueryWord) int {
		documentsOfTheFirst, countedForTheFirst := amountOfDocumentsHeldPerQueryWord[first.word]
		documentsOfTheSecond, countedForTheSecond := amountOfDocumentsHeldPerQueryWord[second.word]
		if countedForTheFirst != countedForTheSecond {
			if countedForTheFirst {
				return -1
			}

			return 1
		}

		return cmp.Compare(documentsOfTheFirst, documentsOfTheSecond)
	}
}
