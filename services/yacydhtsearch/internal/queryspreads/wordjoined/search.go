// Package wordjoined finds documents that match the whole query even when no
// single peer holds every query word. It asks each peer which documents it
// holds for one query word, keeps the documents that came back for every word,
// and asks the peers that hold them for the metadata of those documents. It
// puts each word only to the peers the DHT ring makes responsible for that
// word. Each round stays within the peer calls one query may put: the first
// shares them over the query words, and the second keeps the peers that cover
// the most documents between them. The first round also leaves the second round
// its share of the time the query has left.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerChoice interface {
	ChoosePeersPerQueryWord(
		ctx context.Context,
		queryWords []yacymodel.Hash,
		askablePeers []peerdirectory.AskablePeer,
		peerCallsCeiling int,
	) [][]peerdirectory.AskablePeer
}

type PeerAsks interface {
	AskForHeldDocuments(
		ctx context.Context,
		asks []peerasks.HeldDocumentsAsk,
	) []peerasks.AnsweredHeldDocumentsAsk
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) []peerasks.AnsweredURLMetadataAsk
}

// roundsOfPeerCalls counts the rounds of peer calls one word joined search puts
// one after the other: it asks which documents the peers hold, and then asks
// for the metadata of the documents that joined.
const roundsOfPeerCalls = 2

type Spread struct {
	peerAsks                         PeerAsks
	peerChoice                       PeerChoice
	documentsToAskMetadataForCeiling int
	peerCallsCeiling                 int
	observer                         WordJoinedSearchObserver
}

func New(
	peerAsks PeerAsks,
	peerChoice PeerChoice,
	documentsToAskMetadataForCeiling int,
	peerCallsCeiling int,
	observer WordJoinedSearchObserver,
) Spread {
	return Spread{
		peerAsks:                         peerAsks,
		peerChoice:                       peerChoice,
		documentsToAskMetadataForCeiling: documentsToAskMetadataForCeiling,
		peerCallsCeiling:                 peerCallsCeiling,
		observer:                         observer,
	}
}

func (s Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) [][]searchresult.Item {
	startedAt := time.Now()

	peersPerQueryWord := s.choosePeersPerQueryWord(ctx, query, askablePeers)
	heldDocumentsAsks := asksForTheDocumentsHeldPerQueryWord(query, peersPerQueryWord)
	answeredHeldDocumentsAsks := s.askForHeldDocuments(ctx, heldDocumentsAsks)

	joinedDocuments := joinedDocumentsOf(answeredHeldDocumentsAsks, query.TermHashes())
	documentsToAskMetadataFor := mostHeldDocumentsAmong(
		joinedDocuments,
		answeredHeldDocumentsAsks,
		s.documentsToAskMetadataForCeiling,
	)
	documentsHeldPerPeer := peersCoveringMostDocuments(
		documentsHeldPerPeerAmong(documentsToAskMetadataFor, answeredHeldDocumentsAsks),
		s.peerCallsCeiling,
	)

	urlMetadataAsks := asksForTheJoinedDocuments(documentsHeldPerPeer)
	answeredURLMetadataAsks := s.peerAsks.AskForURLMetadata(ctx, urlMetadataAsks)

	s.observer.WordJoinedSearchPerformed(
		ctx,
		performedWordJoinedSearchFrom(
			query.TermHashes(),
			heldDocumentsAsks,
			answeredHeldDocumentsAsks,
			joinedDocuments,
			documentsToAskMetadataFor,
			answeredURLMetadataAsks,
			time.Since(startedAt),
		),
	)

	return itemsOfEachAnsweredAsk(answeredURLMetadataAsks)
}

func (s Spread) choosePeersPerQueryWord(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) []peersForQueryWord {
	queryWords := query.TermHashes()
	chosenPeers := s.peerChoice.ChoosePeersPerQueryWord(
		ctx, queryWords, askablePeers, s.peerCallsCeiling,
	)

	peersPerQueryWord := make([]peersForQueryWord, 0, len(queryWords))
	for index, queryWord := range queryWords {
		peersPerQueryWord = append(peersPerQueryWord, peersForQueryWord{
			word:  queryWord,
			peers: chosenPeers[index],
		})
	}

	return peersPerQueryWord
}

func asksForTheDocumentsHeldPerQueryWord(
	query searchquery.Query,
	peersPerQueryWord []peersForQueryWord,
) []peerasks.HeldDocumentsAsk {
	var asks []peerasks.HeldDocumentsAsk
	for _, forQueryWord := range peersPerQueryWord {
		for _, peer := range forQueryWord.peers {
			asks = append(asks, peerasks.HeldDocumentsAsk{
				Peer:          peer,
				Word:          forQueryWord.word,
				ExcludedWords: query.ExclusionHashes(),
				Language:      query.Language,
			})
		}
	}

	return asks
}

func (s Spread) askForHeldDocuments(
	ctx context.Context,
	asks []peerasks.HeldDocumentsAsk,
) []peerasks.AnsweredHeldDocumentsAsk {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return s.peerAsks.AskForHeldDocuments(ctx, asks)
	}

	roundCtx, endRound := context.WithTimeout(ctx, time.Until(deadline)/roundsOfPeerCalls)
	defer endRound()

	return s.peerAsks.AskForHeldDocuments(roundCtx, asks)
}

func asksForTheJoinedDocuments(
	documentsHeldPerPeer []documentsHeldByPeer,
) []peerasks.URLMetadataAsk {
	asks := make([]peerasks.URLMetadataAsk, 0, len(documentsHeldPerPeer))
	for _, heldByPeer := range documentsHeldPerPeer {
		asks = append(asks, peerasks.URLMetadataAsk{
			Peer:      heldByPeer.peer,
			Documents: heldByPeer.documents,
		})
	}

	return asks
}

func itemsOfEachAnsweredAsk(
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) [][]searchresult.Item {
	items := make([][]searchresult.Item, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		items = append(items, answeredAsk.Items)
	}

	return items
}
