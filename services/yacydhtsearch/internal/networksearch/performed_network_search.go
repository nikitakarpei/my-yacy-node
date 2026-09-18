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
	rankedDocuments []queryanswers.FoundDocument,
	amountOfAskablePeers int,
	timeSpent time.Duration,
) PerformedNetworkSearch {
	return PerformedNetworkSearch{
		AmountOfAskablePeers:              amountOfAskablePeers,
		AmountOfFoundDocuments:            len(answers.FoundDocuments),
		AmountOfItemsInRanking:            len(rankedDocuments),
		AmountOfRankedItemsCountedByAPeer: amountOfItemsCountedByAPeer(rankedDocuments),
		TimeSpent:                         timeSpent,
	}
}

func amountOfItemsCountedByAPeer(foundDocuments []queryanswers.FoundDocument) int {
	amount := 0
	for _, foundDocument := range foundDocuments {
		if !foundDocument.CountedByAPeer() {
			continue
		}
		amount++
	}

	return amount
}
