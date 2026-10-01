package networksearch

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

type PerformedNetworkSearch struct {
	AmountOfAskablePeers   int
	AmountOfFoundDocuments int
	AmountOfItemsInRanking int
	TimeSpent              time.Duration
}

func performedNetworkSearchFrom(
	findings queryfindings.Findings,
	rankedDocuments []queryfindings.FoundDocument,
	amountOfAskablePeers int,
	timeSpent time.Duration,
) PerformedNetworkSearch {
	return PerformedNetworkSearch{
		AmountOfAskablePeers:   amountOfAskablePeers,
		AmountOfFoundDocuments: len(findings.FoundDocuments),
		AmountOfItemsInRanking: len(rankedDocuments),
		TimeSpent:              timeSpent,
	}
}
