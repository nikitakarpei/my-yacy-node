package documentrelevance

import (
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func addressScoreOf(item peeranswers.AnsweredItem, queryWords []yacymodel.Hash) float64 {
	wordsOfTheHost := wordsOfTheHostOf(item.Metadata.Address)

	amountOfQueryWordsInTheHost := 0
	for _, word := range queryWords {
		if _, inTheHost := wordsOfTheHost[word]; !inTheHost {
			continue
		}
		amountOfQueryWordsInTheHost++
	}

	return float64(amountOfQueryWordsInTheHost)
}

func wordsOfTheHostOf(address string) map[yacymodel.Hash]struct{} {
	readAddress, err := url.Parse(address)
	if err != nil {
		return nil
	}
	spelledWords := yacymodel.WordsIn(readAddress.Hostname())

	wordsOfTheHost := make(map[yacymodel.Hash]struct{}, len(spelledWords))
	for _, spelledWord := range spelledWords {
		wordsOfTheHost[yacymodel.WordHash(spelledWord)] = struct{}{}
	}

	return wordsOfTheHost
}
