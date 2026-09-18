package networksearch

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

type PerformedNetworkSearch struct {
	AmountOfAskablePeers   int
	AmountOfFoundDocuments int
	AmountOfItemsInRanking int
	TimeSpent              time.Duration
}

func performedNetworkSearchFrom(
	answers queryanswers.AnsweredQuery,
	rankedDocuments []queryanswers.FoundDocument,
	amountOfAskablePeers int,
	timeSpent time.Duration,
) PerformedNetworkSearch {
	return PerformedNetworkSearch{
		AmountOfAskablePeers:   amountOfAskablePeers,
		AmountOfFoundDocuments: len(answers.FoundDocuments),
		AmountOfItemsInRanking: len(rankedDocuments),
		TimeSpent:              timeSpent,
	}
}
