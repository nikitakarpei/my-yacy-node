package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

type PerformedWordJoinedSpread struct {
	DiscoveryRound          PerformedDiscoveryRound
	AmountOfJoinedDocuments int
	URLMetadataLookupRound  PerformedURLMetadataLookupRound
	TimeSpent               time.Duration
}

func performedWordJoinedSpreadFrom(
	discoveryRound discoveryRound,
	everyAnswer []wordpartitionasks.ReplicaAnswer,
	joinedDocuments distinctDocuments,
	urlMetadataLookupRound urlMetadataLookupRound,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	return PerformedWordJoinedSpread{
		DiscoveryRound:          performedDiscoveryRoundFrom(discoveryRound, everyAnswer),
		AmountOfJoinedDocuments: len(joinedDocuments),
		URLMetadataLookupRound: performedURLMetadataLookupRoundFrom(
			urlMetadataLookupRound,
			joinedDocuments,
		),
		TimeSpent: timeSpent,
	}
}
