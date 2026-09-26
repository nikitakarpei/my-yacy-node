package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type compoundWordAcrossReplicas struct {
	searchquery.CompoundWord
	queryWordAcrossReplicas
}

func compoundWordsAcrossReplicasFrom(
	compoundWords []searchquery.CompoundWord,
	settledAsks settledAsks,
	partitions yacymodel.DHTRingPartitions,
) []compoundWordAcrossReplicas {
	var compoundWordsAcrossReplicas []compoundWordAcrossReplicas
	for _, compoundWord := range compoundWords {
		if !slices.ContainsFunc(
			settledAsks,
			func(settledAsk wordpartitionasks.SettledAsk) bool {
				return settledAsk.Word == compoundWord.Hash()
			},
		) {
			continue
		}
		compoundWordsAcrossReplicas = append(
			compoundWordsAcrossReplicas,
			compoundWordAcrossReplicas{
				CompoundWord: compoundWord,
				queryWordAcrossReplicas: queryWordAcrossReplicasFrom(
					compoundWord.Hash(),
					settledAsks,
					partitions,
				),
			},
		)
	}

	return compoundWordsAcrossReplicas
}
