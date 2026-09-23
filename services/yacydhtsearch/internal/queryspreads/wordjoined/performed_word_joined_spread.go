package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

type PerformedWordJoinedSpread struct {
	PeerStandings    []peerjudgements.PeerStanding
	DiscoveryRound   PerformedDiscoveryRound
	CrossCheckRound  PerformedCrossCheckRound
	URLMetadataRound PerformedURLMetadataRound
	TimeSpent        time.Duration
}

//nolint:revive // argument-limit: the report takes the three rounds, the judging and the join
func performedWordJoinedSpreadFrom(
	discoveryRound discoveryRound,
	crossCheckRound crossCheckRound,
	peerStandings peerjudgements.PeerStandings,
	judgedPeers []peerjudgements.JudgedPeer,
	joinedDocuments distinctDocuments,
	urlMetadataRound urlMetadataRound,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	return PerformedWordJoinedSpread{
		PeerStandings: peerStandings,
		DiscoveryRound: performedDiscoveryRoundFrom(
			discoveryRound,
		),
		CrossCheckRound: performedCrossCheckRoundFrom(
			crossCheckRound,
			discoveryRound,
			judgedPeers,
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
