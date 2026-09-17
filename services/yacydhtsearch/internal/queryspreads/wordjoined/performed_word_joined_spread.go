package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PerformedWordJoinedSpread struct {
	MatchedAndHeldDocumentsRound PerformedMatchedAndHeldDocumentsRound
	CrossCheckedDocumentsRound   PerformedCrossCheckedDocumentsRound
	URLMetadataRound             PerformedURLMetadataRound
	TimeSpent                    time.Duration
}

func performedWordJoinedSpreadFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	crossCheckedDocumentsRound crossCheckedDocumentsRound,
	joinedDocuments distinctDocuments,
	urlMetadataRound urlMetadataRound,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	return PerformedWordJoinedSpread{
		MatchedAndHeldDocumentsRound: performedMatchedAndHeldDocumentsRoundFrom(
			matchedAndHeldDocumentsRound,
		),
		CrossCheckedDocumentsRound: performedCrossCheckedDocumentsRoundFrom(
			crossCheckedDocumentsRound,
			matchedAndHeldDocumentsRound,
			joinedDocuments,
		),
		URLMetadataRound: performedURLMetadataRoundFrom(urlMetadataRound, joinedDocuments),
		TimeSpent:        timeSpent,
	}
}

func amountOfPeersAcross[Ask any](
	asks []Ask,
	peerOfAsk func(Ask) peerdirectory.AskablePeer,
) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, ask := range asks {
		peers[peerOfAsk(ask)] = struct{}{}
	}

	return len(peers)
}
