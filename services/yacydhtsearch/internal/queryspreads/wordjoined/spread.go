// Package wordjoined finds documents that match the whole query even when no
// single peer holds every query word. It asks each peer which documents it
// holds for one query word and which documents it matches for that word, keeps
// the documents that came back for every word, and asks the peers that hold
// them for the metadata of the joined documents that came back without it.
// It puts each word only to the peers the DHT ring makes responsible for that
// word, and asks as many peers for each word as hold that word. The second
// round asks that same amount of peers, and keeps the peers that cover the most
// documents between them. The first round leaves the second round its share of
// the time the query has left.
package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerChoice interface {
	ChoosePeersPerQueryWord(
		ctx context.Context,
		queryWords []yacymodel.Hash,
		askablePeers []peerdirectory.AskablePeer,
		peersHoldingOneWord int,
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

type Spread struct {
	peerAsks                 PeerAsks
	peerChoice               PeerChoice
	metadataDocumentsCeiling int
	peerItemsCeiling         int
	peersHoldingOneWord      int
	observer                 WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the ceilings one word joined spread stays within
func New(
	peerAsks PeerAsks,
	peerChoice PeerChoice,
	metadataDocumentsCeiling int,
	peerItemsCeiling int,
	peersHoldingOneWord int,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		peerAsks:                 peerAsks,
		peerChoice:               peerChoice,
		metadataDocumentsCeiling: metadataDocumentsCeiling,
		peerItemsCeiling:         peerItemsCeiling,
		peersHoldingOneWord:      peersHoldingOneWord,
		observer:                 observer,
	}
}

func (s Spread) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) peeranswers.AnsweredQuery {
	startedAt := time.Now()

	chosenPeersPerQueryWord := s.peerChoice.ChoosePeersPerQueryWord(
		ctx, query.TermHashes(), askablePeers, s.peersHoldingOneWord,
	)
	heldDocumentsAsks, answeredHeldDocumentsAsks := s.askForHeldDocuments(
		ctx, query, chosenPeersPerQueryWord,
	)
	joinedDocuments := joinedDocumentsOf(answeredHeldDocumentsAsks, query.TermHashes())
	itemsInTheOrderOfEachPeerRanking := itemsInTheOrderOfEachPeerRankingAmong(
		answeredHeldDocumentsAsks, joinedDocuments,
	)
	documentsWithoutMetadata := joinedDocumentsWithoutMetadata(
		joinedDocuments, itemsInTheOrderOfEachPeerRanking,
	)
	urlMetadataAsks, answeredURLMetadataAsks := s.askForURLMetadata(
		ctx, documentsWithoutMetadata, answeredHeldDocumentsAsks,
	)

	s.observer.WordJoinedSpreadPerformed(
		ctx,
		performedWordJoinedSpreadFrom(
			query.TermHashes(),
			heldDocumentsAsks,
			answeredHeldDocumentsAsks,
			joinedDocuments,
			documentsWithoutMetadata,
			urlMetadataAsks,
			answeredURLMetadataAsks,
			time.Since(startedAt),
		),
	)

	return answeredQueryFrom(
		itemsInTheOrderOfEachPeerRanking,
		answeredURLMetadataAsks,
		answeredHeldDocumentsAsks,
		query.TermHashes(),
	)
}
