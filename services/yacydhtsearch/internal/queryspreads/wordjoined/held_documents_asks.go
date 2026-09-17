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
	for _, peersShortOfOneQueryWord := range peersShortOfEachQueryWordAmong(
		peersShortOfTheirQueryWord,
	) {
		asksOfTheQueryWord, documentsPastTheCeilingOfTheQueryWord := heldDocumentsAsksDealtAcross(
			peersWithoutAnAskAmong(peersShortOfOneQueryWord, asks),
			documentsMostHeldFirst,
			heldDocumentsCeiling,
		)
		asks = append(asks, asksOfTheQueryWord...)
		for _, document := range documentsPastTheCeilingOfTheQueryWord {
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

func peersShortOfEachQueryWordAmong(
	peersShortOfTheirQueryWord []peerShortOfAQueryWord,
) [][]peerShortOfAQueryWord {
	peersShortOfEachQueryWord := make([][]peerShortOfAQueryWord, 0, len(peersShortOfTheirQueryWord))
	placeOfEachQueryWord := map[yacymodel.Hash]int{}
	for _, peerShortOfItsQueryWord := range peersShortOfTheirQueryWord {
		place, grouped := placeOfEachQueryWord[peerShortOfItsQueryWord.word]
		if !grouped {
			place = len(peersShortOfEachQueryWord)
			placeOfEachQueryWord[peerShortOfItsQueryWord.word] = place
			peersShortOfEachQueryWord = append(peersShortOfEachQueryWord, nil)
		}
		peersShortOfEachQueryWord[place] = append(
			peersShortOfEachQueryWord[place], peerShortOfItsQueryWord,
		)
	}

	return peersShortOfEachQueryWord
}

func peersWithoutAnAskAmong(
	peersShortOfOneQueryWord []peerShortOfAQueryWord,
	asks []peerasks.HeldDocumentsAsk,
) []peerShortOfAQueryWord {
	keptPeers := make([]peerShortOfAQueryWord, 0, len(peersShortOfOneQueryWord))
	for _, peerShortOfItsQueryWord := range peersShortOfOneQueryWord {
		if slices.ContainsFunc(asks, func(ask peerasks.HeldDocumentsAsk) bool {
			return ask.Peer.Hash == peerShortOfItsQueryWord.peer.Hash
		}) {
			continue
		}
		keptPeers = append(keptPeers, peerShortOfItsQueryWord)
	}

	return keptPeers
}

func heldDocumentsAsksDealtAcross(
	peersShortOfOneQueryWord []peerShortOfAQueryWord,
	documentsMostHeldFirst []yacymodel.URLHash,
	heldDocumentsCeiling int,
) ([]peerasks.HeldDocumentsAsk, []yacymodel.URLHash) {
	documentsDealtToEachPeer := make([][]yacymodel.URLHash, len(peersShortOfOneQueryWord))
	documentsPastTheCeiling := make([]yacymodel.URLHash, 0, len(documentsMostHeldFirst))
	placeOfTheNextTurn := 0
	for _, document := range documentsMostHeldFirst {
		place, dealt := placeOfThePeerTakingTheDocument(
			peersShortOfOneQueryWord,
			documentsDealtToEachPeer,
			document,
			placeOfTheNextTurn,
			heldDocumentsCeiling,
		)
		if !dealt {
			if !everyPeerNamedTheDocument(peersShortOfOneQueryWord, document) {
				documentsPastTheCeiling = append(documentsPastTheCeiling, document)
			}

			continue
		}
		documentsDealtToEachPeer[place] = append(documentsDealtToEachPeer[place], document)
		placeOfTheNextTurn = (place + 1) % len(peersShortOfOneQueryWord)
	}

	return heldDocumentsAsksNamingTheDealtDocuments(
		peersShortOfOneQueryWord, documentsDealtToEachPeer,
	), documentsPastTheCeiling
}

func placeOfThePeerTakingTheDocument(
	peersShortOfOneQueryWord []peerShortOfAQueryWord,
	documentsDealtToEachPeer [][]yacymodel.URLHash,
	document yacymodel.URLHash,
	placeOfTheNextTurn int,
	heldDocumentsCeiling int,
) (int, bool) {
	for turn := range peersShortOfOneQueryWord {
		place := (placeOfTheNextTurn + turn) % len(peersShortOfOneQueryWord)
		if len(documentsDealtToEachPeer[place]) >= heldDocumentsCeiling ||
			slices.Contains(peersShortOfOneQueryWord[place].documentsTheAnswerNamed, document) {
			continue
		}

		return place, true
	}

	return 0, false
}

func everyPeerNamedTheDocument(
	peersShortOfOneQueryWord []peerShortOfAQueryWord,
	document yacymodel.URLHash,
) bool {
	for _, peerShortOfItsQueryWord := range peersShortOfOneQueryWord {
		if !slices.Contains(peerShortOfItsQueryWord.documentsTheAnswerNamed, document) {
			return false
		}
	}

	return true
}

func heldDocumentsAsksNamingTheDealtDocuments(
	peersShortOfOneQueryWord []peerShortOfAQueryWord,
	documentsDealtToEachPeer [][]yacymodel.URLHash,
) []peerasks.HeldDocumentsAsk {
	asks := make([]peerasks.HeldDocumentsAsk, 0, len(peersShortOfOneQueryWord))
	for place, documentsDealtToOnePeer := range documentsDealtToEachPeer {
		if len(documentsDealtToOnePeer) == 0 {
			continue
		}
		asks = append(asks, peerasks.HeldDocumentsAsk{
			Peer:      peersShortOfOneQueryWord[place].peer,
			Word:      peersShortOfOneQueryWord[place].word,
			Documents: documentsDealtToOnePeer,
		})
	}

	return asks
}
