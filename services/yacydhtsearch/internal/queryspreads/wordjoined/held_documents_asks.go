package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func heldDocumentsAsksFor(
	shortQueryWords []shortQueryWord,
	anchorDocuments map[yacymodel.URLHash]struct{},
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	heldDocumentsCeiling int,
) ([]peerasks.HeldDocumentsAsk, int) {
	documentsMostHeldFirst := documentsMostHeldFirstAmong(anchorDocuments, answeredAsks)
	peersShortOfTheirQueryWord := peersShortOfTheirQueryWordAcross(shortQueryWords, answeredAsks)
	asks := make([]peerasks.HeldDocumentsAsk, 0, len(peersShortOfTheirQueryWord))
	documentsPastTheCeiling := map[yacymodel.URLHash]struct{}{}
	askedPeers := map[yacymodel.Hash]struct{}{}
	for _, peerShortOfItsQueryWord := range peersShortOfTheirQueryWord {
		if _, asked := askedPeers[peerShortOfItsQueryWord.peer.Hash]; asked {
			continue
		}
		documentsToName, documentsLeftOut := documentsWithinTheCeiling(
			documentsMostHeldFirst,
			peerShortOfItsQueryWord.documentsTheAnswerNamed,
			heldDocumentsCeiling,
		)
		if len(documentsToName) == 0 {
			continue
		}
		askedPeers[peerShortOfItsQueryWord.peer.Hash] = struct{}{}
		asks = append(asks, peerasks.HeldDocumentsAsk{
			Peer:      peerShortOfItsQueryWord.peer,
			Word:      peerShortOfItsQueryWord.word,
			Documents: documentsToName,
		})
		for _, document := range documentsLeftOut {
			documentsPastTheCeiling[document] = struct{}{}
		}
	}

	return asks, len(documentsPastTheCeiling)
}

type peerShortOfAQueryWord struct {
	peer                    peerdirectory.AskablePeer
	word                    yacymodel.Hash
	documentsTheAnswerNamed []yacymodel.URLHash
}

func peersShortOfTheirQueryWordAcross(
	shortQueryWords []shortQueryWord,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []peerShortOfAQueryWord {
	answerOfEachPeerPerQueryWord := answerOfEachPeerPerQueryWordOf(answeredAsks)
	peersShortOfTheirQueryWord := make([]peerShortOfAQueryWord, 0, len(shortQueryWords))
	for _, shortQueryWord := range shortQueryWords {
		for _, peer := range shortQueryWord.peers {
			answeredAsk, answered := answerOfEachPeerPerQueryWord[answeringPeerOfAQueryWord{
				peer: peer.Hash, word: shortQueryWord.word,
			}]
			if answered && answerIsComplete(answeredAsk) {
				continue
			}
			peersShortOfTheirQueryWord = append(peersShortOfTheirQueryWord, peerShortOfAQueryWord{
				peer:                    peer,
				word:                    shortQueryWord.word,
				documentsTheAnswerNamed: answeredAsk.DocumentsHeldForTheWord,
			})
		}
	}

	return peersShortOfTheirQueryWord
}

type answeringPeerOfAQueryWord struct {
	peer yacymodel.Hash
	word yacymodel.Hash
}

func answerOfEachPeerPerQueryWordOf(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) map[answeringPeerOfAQueryWord]peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	answerOfEachPeerPerQueryWord := make(
		map[answeringPeerOfAQueryWord]peerasks.AnsweredMatchedAndHeldDocumentsAsk,
		len(answeredAsks),
	)
	for _, answeredAsk := range answeredAsks {
		answerOfEachPeerPerQueryWord[answeringPeerOfAQueryWord{
			peer: answeredAsk.Ask.Peer.Hash, word: answeredAsk.Ask.Word,
		}] = answeredAsk
	}

	return answerOfEachPeerPerQueryWord
}

func documentsWithinTheCeiling(
	documentsMostHeldFirst []yacymodel.URLHash,
	documentsTheAnswerNamed []yacymodel.URLHash,
	heldDocumentsCeiling int,
) ([]yacymodel.URLHash, []yacymodel.URLHash) {
	documentsToName := make([]yacymodel.URLHash, 0, len(documentsMostHeldFirst))
	for _, document := range documentsMostHeldFirst {
		if slices.Contains(documentsTheAnswerNamed, document) {
			continue
		}
		documentsToName = append(documentsToName, document)
	}
	if len(documentsToName) <= heldDocumentsCeiling {
		return documentsToName, nil
	}

	return documentsToName[:heldDocumentsCeiling], documentsToName[heldDocumentsCeiling:]
}
