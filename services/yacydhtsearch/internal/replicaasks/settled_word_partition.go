package replicaasks

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"

type SettledWordPartition struct {
	AskOutcomes []PlacedAskOutcome
}

type PlacedAskOutcome struct {
	PlaceInTheRun int
	AskOutcome    peerasks.SearchDocumentsAskOutcome
}
