package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type compoundWordAcrossReplicas struct {
	searchquery.CompoundWord
	queryWordAcrossReplicas
}

func compoundWordsAcrossReplicasFrom(
	compoundWords []searchquery.CompoundWord,
	askOutcomes peerasks.WordAbstractAskOutcomes,
	partitions yacymodel.DHTRingPartitions,
) []compoundWordAcrossReplicas {
	var compoundWordsAcrossReplicas []compoundWordAcrossReplicas
	for _, compoundWord := range compoundWords {
		if !slices.ContainsFunc(
			askOutcomes,
			func(askOutcome peerasks.WordAbstractAskOutcome) bool {
				return askOutcome.Ask.Word == compoundWord.Hash
			},
		) {
			continue
		}
		compoundWordsAcrossReplicas = append(
			compoundWordsAcrossReplicas,
			compoundWordAcrossReplicas{
				CompoundWord: compoundWord,
				queryWordAcrossReplicas: queryWordAcrossReplicasFrom(
					compoundWord.Hash,
					askOutcomes,
					partitions,
				),
			},
		)
	}

	return compoundWordsAcrossReplicas
}
