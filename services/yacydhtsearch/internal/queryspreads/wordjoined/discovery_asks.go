package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type discoveryAsks []wordpartitionasks.Ask

func discoveryAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) discoveryAsks {
	var asks discoveryAsks
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeersOfPartition := range chosenPeersOfQueryWord.ChosenPeersPerPartition() {
			asks = append(asks, wordpartitionasks.Ask{
				Word:            chosenPeersOfQueryWord.QueryWord,
				Partition:       chosenPeersOfPartition.Partition,
				ReplicasInOrder: chosenPeersOfPartition.Peers,
				ExcludedWords:   query.ExclusionHashes(),
				Language:        query.Language,
			})
		}
	}

	return asks
}

func (asks discoveryAsks) ofWords(words []yacymodel.Hash) discoveryAsks {
	var asksOfTheWords discoveryAsks
	for _, ask := range asks {
		if !slices.Contains(words, ask.Word) {
			continue
		}
		asksOfTheWords = append(asksOfTheWords, ask)
	}

	return asksOfTheWords
}

func (asks discoveryAsks) ofWordsIn(words []yacymodel.Hash, partition uint) discoveryAsks {
	var asksOfTheWords discoveryAsks
	for _, ask := range asks.ofWords(words) {
		if ask.Partition != partition {
			continue
		}
		asksOfTheWords = append(asksOfTheWords, ask)
	}

	return asksOfTheWords
}

func (asks discoveryAsks) forDocumentsToMatch(documentsToMatch []yacymodel.URLHash) discoveryAsks {
	asksForTheDocuments := make(discoveryAsks, 0, len(asks))
	for _, ask := range asks {
		ask.DocumentsToMatch = documentsToMatch
		asksForTheDocuments = append(asksForTheDocuments, ask)
	}

	return asksForTheDocuments
}
