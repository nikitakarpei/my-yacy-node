package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type compoundWordAcrossReplicas struct {
	searchquery.CompoundWord
	queryWordAcrossReplicas
}

func compoundWordsAcrossReplicasFrom(
	compoundWords []searchquery.CompoundWord,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	partitions yacymodel.DHTRingPartitions,
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) []compoundWordAcrossReplicas {
	var compoundWordsAcrossReplicas []compoundWordAcrossReplicas
	for _, compoundWord := range compoundWords {
		chosenPeersOfCompoundWord, asked := chosenPeersOf(
			compoundWord.Hash,
			chosenPeersPerQueryWord,
		)
		if !asked {
			continue
		}
		compoundWordsAcrossReplicas = append(
			compoundWordsAcrossReplicas,
			compoundWordAcrossReplicas{
				CompoundWord: compoundWord,
				queryWordAcrossReplicas: queryWordAcrossReplicasFrom(
					chosenPeersOfCompoundWord, partitions, answeredAsks,
				),
			},
		)
	}

	return compoundWordsAcrossReplicas
}
