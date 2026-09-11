// Package wordjoined finds documents that match the whole query even when no
// single peer holds every query word. It asks each peer which documents it
// holds for one query word and which documents it matches for that word, keeps
// the documents that came back for every word, and asks the peers that hold
// them for the metadata of the joined documents that came back without it.
// It puts each word only to the peers the DHT ring makes responsible for that
// word. Each round stays within the peer calls one query may put: the first
// shares them over the query words, and the second keeps the peers that cover
// the most documents between them. The first round also leaves the second round
// its share of the time the query has left.
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

type Spread struct {
	peerAsks                 PeerAsks
	peerChoice               PeerChoice
	metadataDocumentsCeiling int
	peerItemsCeiling         int
	peerCallsCeiling         int
	observer                 WordJoinedSpreadObserver
}

//nolint:revive // argument-limit: the ceilings one word joined spread stays within
func New(
	peerAsks PeerAsks,
	peerChoice PeerChoice,
	metadataDocumentsCeiling int,
	peerItemsCeiling int,
	peerCallsCeiling int,
	observer WordJoinedSpreadObserver,
) Spread {
	return Spread{
		peerAsks:                 peerAsks,
		peerChoice:               peerChoice,
		metadataDocumentsCeiling: metadataDocumentsCeiling,
		peerItemsCeiling:         peerItemsCeiling,
		peerCallsCeiling:         peerCallsCeiling,
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
		ctx, query.TermHashes(), askablePeers, s.peerCallsCeiling,
	)
	heldDocumentsAsks, answeredHeldDocumentsAsks := s.askForHeldDocuments(
		ctx, query, chosenPeersPerQueryWord,
	)
	joinedDocuments := joinedDocumentsOf(answeredHeldDocumentsAsks, query.TermHashes())
	itemsInTheOrderOfEachAnswer := itemsInTheOrderOfEachAnswerAmong(
		answeredHeldDocumentsAsks, joinedDocuments,
	)
	documentsWithoutMetadata := joinedDocumentsWithoutMetadata(
		joinedDocuments, itemsInTheOrderOfEachAnswer,
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
		itemsInTheOrderOfEachAnswer,
		answeredURLMetadataAsks,
		answeredHeldDocumentsAsks,
		query.TermHashes(),
	)
}
