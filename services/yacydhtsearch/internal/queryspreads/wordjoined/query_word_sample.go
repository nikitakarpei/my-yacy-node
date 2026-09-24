package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryWordSample struct {
	partition         uint
	sampledQueryWords []sampledQueryWord
}

type sampledQueryWord struct {
	word              yacymodel.Hash
	amountOfDocuments int
}

func queryWordSampleIn(
	partition uint,
	queryWords []yacymodel.Hash,
	askOutcomes peerasks.SearchDocumentsAskOutcomes,
	partitions yacymodel.DHTRingPartitions,
) queryWordSample {
	sample := queryWordSample{partition: partition}
	for _, queryWord := range queryWords {
		replicas := queryWordAcrossReplicasFrom(queryWord, askOutcomes, partitions).
			replicasPerPartition[partition]
		amountOfDocuments, complete := amountOfDocumentsSampledIn(
			replicas, partition, partitions,
		).Get()
		if !complete {
			continue
		}
		sample.sampledQueryWords = append(sample.sampledQueryWords, sampledQueryWord{
			word:              queryWord,
			amountOfDocuments: amountOfDocuments,
		})
	}

	return sample
}

func amountOfDocumentsSampledIn(
	replicas []wordReplica,
	partition uint,
	partitions yacymodel.DHTRingPartitions,
) yacymodel.Optional[int] {
	documentsInThePartition := distinctDocuments{}
	complete := false
	for _, replica := range replicas {
		answer, answered := replica.answer.Get()
		if !answered || !replica.hasACompleteAbstract() {
			continue
		}
		complete = true
		for _, document := range answer.Abstract {
			if partitions.PartitionOf(document) != partition {
				continue
			}
			documentsInThePartition.add(document)
		}
	}
	if !complete {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(len(documentsInThePartition))
}

func (sample queryWordSample) rarestQueryWord() yacymodel.Optional[yacymodel.Hash] {
	if len(sample.sampledQueryWords) == 0 {
		return yacymodel.None[yacymodel.Hash]()
	}
	rarestSampledQueryWord := sample.sampledQueryWords[0]
	for _, otherSampledQueryWord := range sample.sampledQueryWords[1:] {
		if otherSampledQueryWord.amountOfDocuments < rarestSampledQueryWord.amountOfDocuments {
			rarestSampledQueryWord = otherSampledQueryWord
		}
	}

	return yacymodel.Some(rarestSampledQueryWord.word)
}

func (sample queryWordSample) amountOfSampledQueryWords() int {
	return len(sample.sampledQueryWords)
}
