package documentasks

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type asks []wordpartitionasks.Ask

func asksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) asks {
	var plannedAsks asks
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeersOfPartition := range chosenPeersOfQueryWord.ChosenPeersPerPartition() {
			plannedAsks = append(plannedAsks, wordpartitionasks.Ask{
				Word:            chosenPeersOfQueryWord.QueryWord,
				Partition:       chosenPeersOfPartition.Partition,
				ReplicasInOrder: chosenPeersOfPartition.Peers,
				ExcludedWords:   query.ExclusionHashes(),
				Language:        query.Language,
			})
		}
	}

	return plannedAsks
}

func (plannedAsks asks) ofWords(words []yacymodel.Hash) asks {
	var asksOfTheWords asks
	for _, ask := range plannedAsks {
		if !slices.Contains(words, ask.Word) {
			continue
		}
		asksOfTheWords = append(asksOfTheWords, ask)
	}

	return asksOfTheWords
}

func (plannedAsks asks) ofWordsIn(words []yacymodel.Hash, partition uint) asks {
	var asksOfTheWords asks
	for _, ask := range plannedAsks.ofWords(words) {
		if ask.Partition != partition {
			continue
		}
		asksOfTheWords = append(asksOfTheWords, ask)
	}

	return asksOfTheWords
}

func (plannedAsks asks) forDocumentsToMatch(documentsToMatch []yacymodel.URLHash) asks {
	asksForTheDocuments := make(asks, 0, len(plannedAsks))
	for _, ask := range plannedAsks {
		ask.DocumentsToMatch = documentsToMatch
		asksForTheDocuments = append(asksForTheDocuments, ask)
	}

	return asksForTheDocuments
}
