package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func rarestQueryWordIn(
	sampledPartition uint,
	queryWords []queryWordAcrossReplicas,
	partitions yacymodel.DHTRingPartitions,
) yacymodel.Optional[yacymodel.Hash] {
	rarestQueryWord := yacymodel.None[yacymodel.Hash]()
	fewestDocuments := 0
	for _, queryWord := range queryWords {
		amountOfDocuments, complete := queryWord.amountOfDocumentsIn(sampledPartition, partitions).
			Get()
		if !complete || rarestQueryWord.Present() && amountOfDocuments >= fewestDocuments {
			continue
		}
		rarestQueryWord = yacymodel.Some(queryWord.word)
		fewestDocuments = amountOfDocuments
	}

	return rarestQueryWord
}

func amountOfQueryWordsSampledIn(
	sampledPartition uint,
	queryWords []queryWordAcrossReplicas,
	partitions yacymodel.DHTRingPartitions,
) int {
	amountOfQueryWordsSampled := 0
	for _, queryWord := range queryWords {
		if queryWord.amountOfDocumentsIn(sampledPartition, partitions).Present() {
			amountOfQueryWordsSampled++
		}
	}

	return amountOfQueryWordsSampled
}
