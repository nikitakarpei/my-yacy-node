package networksearch

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

type PerformedNetworkSearch struct {
	AmountOfAskablePeers              int
	AmountOfFoundDocuments            int
	AmountOfItemsInRanking            int
	AmountOfRankedItemsCountedByAPeer int
	TimeSpent                         time.Duration
}

func performedNetworkSearchFrom(
	answers queryanswers.AnsweredQuery,
	rankedItems []queryanswers.FoundDocument,
	amountOfAskablePeers int,
	timeSpent time.Duration,
) PerformedNetworkSearch {
	return PerformedNetworkSearch{
		AmountOfAskablePeers:              amountOfAskablePeers,
		AmountOfFoundDocuments:            len(answers.FoundDocuments),
		AmountOfItemsInRanking:            len(rankedItems),
		AmountOfRankedItemsCountedByAPeer: amountOfItemsCountedByAPeer(rankedItems),
		TimeSpent:                         timeSpent,
	}
}

func amountOfItemsCountedByAPeer(items []queryanswers.FoundDocument) int {
	amount := 0
	for _, item := range items {
		if !item.CountedByAPeer() {
			continue
		}
		amount++
	}

	return amount
}
