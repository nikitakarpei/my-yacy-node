package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func rarestQueryWordIn(
	sampledPartition uint,
	queryWords []wordAcrossReplicas,
	partitions yacymodel.DHTRingPartitions,
) yacymodel.Optional[yacymodel.Hash] {
	rarestQueryWord := yacymodel.None[yacymodel.Hash]()
	fewestDocuments := 0
	for _, queryWord := range queryWords {
		amountOfDocuments, complete := queryWord.sampleIn(sampledPartition, partitions).
			Get()
		if !complete || rarestQueryWord.Present() && amountOfDocuments >= fewestDocuments {
			continue
		}
		rarestQueryWord = yacymodel.Some(queryWord.hash)
		fewestDocuments = amountOfDocuments
	}

	return rarestQueryWord
}

func amountOfQueryWordsWithASampleIn(
	sampledPartition uint,
	queryWords []wordAcrossReplicas,
	partitions yacymodel.DHTRingPartitions,
) int {
	amountOfQueryWordsWithASample := 0
	for _, queryWord := range queryWords {
		if queryWord.sampleIn(sampledPartition, partitions).Present() {
			amountOfQueryWordsWithASample++
		}
	}

	return amountOfQueryWordsWithASample
}
