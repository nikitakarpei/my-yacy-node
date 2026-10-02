package wordholdings

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type holdingsOfCompoundWord struct {
	searchquery.CompoundWord
	holdingsOfWord
}

func holdingsOfEachCompoundWordFrom(
	compoundWords []searchquery.CompoundWord,
	settledAsks []wordpartitionasks.SettledAsk,
	partitions yacymodel.DHTRingPartitions,
) []holdingsOfCompoundWord {
	var holdingsOfEachCompoundWord []holdingsOfCompoundWord
	for _, compoundWord := range compoundWords {
		if !slices.ContainsFunc(
			settledAsks,
			func(settledAsk wordpartitionasks.SettledAsk) bool {
				return settledAsk.Word == compoundWord.Hash()
			},
		) {
			continue
		}
		holdingsOfEachCompoundWord = append(
			holdingsOfEachCompoundWord,
			holdingsOfCompoundWord{
				CompoundWord: compoundWord,
				holdingsOfWord: holdingsOfWordFrom(
					compoundWord.Hash(),
					settledAsks,
					partitions,
				),
			},
		)
	}

	return holdingsOfEachCompoundWord
}
