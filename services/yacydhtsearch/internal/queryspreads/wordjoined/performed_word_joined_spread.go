package wordjoined

import "time"

type PerformedWordJoinedSpread struct {
	DiscoveryRound          PerformedDiscoveryRound
	AmountOfJoinedDocuments int
	URLMetadataLookupRound  PerformedURLMetadataLookupRound
	TimeSpent               time.Duration
}

func performedWordJoinedSpreadFrom(
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
	urlMetadataLookupRound urlMetadataLookupRound,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	return PerformedWordJoinedSpread{
		DiscoveryRound: performedDiscoveryRoundFrom(
			discoveryRound,
		),
		AmountOfJoinedDocuments: len(joinedDocuments),
		URLMetadataLookupRound:  performedURLMetadataLookupRoundFrom(urlMetadataLookupRound),
		TimeSpent:               timeSpent,
	}
}
