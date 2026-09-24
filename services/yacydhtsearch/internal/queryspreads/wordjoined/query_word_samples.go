package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryWordSamples struct {
	partition         uint
	sampledQueryWords []sampledQueryWord
}

type sampledQueryWord struct {
	word              yacymodel.Hash
	amountOfDocuments int
}

func queryWordSamplesIn(
	partition uint,
	queryWords []yacymodel.Hash,
	askOutcomes peerasks.SearchDocumentsAskOutcomes,
	partitions yacymodel.DHTRingPartitions,
) queryWordSamples {
	samples := queryWordSamples{partition: partition}
	for _, queryWord := range queryWords {
		wordPartition := queryWordAcrossReplicasFrom(queryWord, askOutcomes, partitions).
			wordPartitions()[partition]
		amountOfDocuments, sampled := sampleFrom(wordPartition, partitions).Get()
		if !sampled {
			continue
		}
		samples.sampledQueryWords = append(samples.sampledQueryWords, sampledQueryWord{
			word:              queryWord,
			amountOfDocuments: amountOfDocuments,
		})
	}

	return samples
}

func sampleFrom(
	wordPartition wordPartition,
	partitions yacymodel.DHTRingPartitions,
) yacymodel.Optional[int] {
	documentsInThePartition := distinctDocuments{}
	sampled := false
	for _, replica := range wordPartition.replicas {
		answer, answered := replica.answer.Get()
		if !answered || !replica.hasACompleteAbstract() {
			continue
		}
		sampled = true
		for _, document := range answer.Abstract {
			if partitions.PartitionOf(document) != wordPartition.partition {
				continue
			}
			documentsInThePartition.add(document)
		}
	}
	if !sampled {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(len(documentsInThePartition))
}

func (samples queryWordSamples) rarestQueryWord() yacymodel.Optional[yacymodel.Hash] {
	if len(samples.sampledQueryWords) == 0 {
		return yacymodel.None[yacymodel.Hash]()
	}
	rarestSampledQueryWord := samples.sampledQueryWords[0]
	for _, otherSampledQueryWord := range samples.sampledQueryWords[1:] {
		if otherSampledQueryWord.amountOfDocuments < rarestSampledQueryWord.amountOfDocuments {
			rarestSampledQueryWord = otherSampledQueryWord
		}
	}

	return yacymodel.Some(rarestSampledQueryWord.word)
}

func (samples queryWordSamples) amountOfSampledQueryWords() int {
	return len(samples.sampledQueryWords)
}
