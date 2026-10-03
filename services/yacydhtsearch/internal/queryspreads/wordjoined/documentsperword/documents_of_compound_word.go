package documentsperword

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentsOfCompoundWord struct {
	searchquery.CompoundWord
	documentsOfWord
}

func documentsOfEachCompoundWordFrom(
	compoundWords []searchquery.CompoundWord,
	settledAsks []wordpartitionasks.SettledAsk,
	partitions yacymodel.DHTRingPartitions,
) []documentsOfCompoundWord {
	var documentsOfEachCompoundWord []documentsOfCompoundWord
	for _, compoundWord := range compoundWords {
		if !slices.ContainsFunc(
			settledAsks,
			func(settledAsk wordpartitionasks.SettledAsk) bool {
				return settledAsk.Word == compoundWord.Hash()
			},
		) {
			continue
		}
		documentsOfEachCompoundWord = append(
			documentsOfEachCompoundWord,
			documentsOfCompoundWord{
				CompoundWord: compoundWord,
				documentsOfWord: documentsOfWordFrom(
					compoundWord.Hash(),
					settledAsks,
					partitions,
				),
			},
		)
	}

	return documentsOfEachCompoundWord
}
