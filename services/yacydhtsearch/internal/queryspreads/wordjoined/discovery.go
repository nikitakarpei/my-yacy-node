package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type discovery struct {
	askRun                    *askRun
	asks                      discoveryAsks
	query                     searchquery.Query
	partitions                yacymodel.DHTRingPartitions
	documentsToMatchCeiling   int
	otherWordAsksPerPartition map[uint]OtherWordAsks
}

func discoveryOver(
	askRun *askRun,
	asks discoveryAsks,
	query searchquery.Query,
	partitions yacymodel.DHTRingPartitions,
	documentsToMatchCeiling int,
) *discovery {
	return &discovery{
		askRun:                    askRun,
		asks:                      asks,
		query:                     query,
		partitions:                partitions,
		documentsToMatchCeiling:   documentsToMatchCeiling,
		otherWordAsksPerPartition: map[uint]OtherWordAsks{},
	}
}

func (discovery *discovery) askFromTheSampleIn(sampledPartition uint) discoveryRound {
	leadingQueryWordFromTheSample := discovery.takeTheSampleIn(sampledPartition)
	discovery.askTheRestAfter(leadingQueryWordFromTheSample)
	discovery.askRun.finish()

	return discovery.roundFrom(sampledPartition, leadingQueryWordFromTheSample)
}

func (discovery *discovery) takeTheSampleIn(
	sampledPartition uint,
) yacymodel.Optional[yacymodel.Hash] {
	queryWords := discovery.query.WordHashes()
	asksOfTheSample := discovery.asks.ofWordsIn(queryWords, sampledPartition)
	discovery.askRun.put(asksOfTheSample)
	discovery.askRun.readUntilSettled(asksOfTheSample)

	return rarestQueryWordIn(
		sampledPartition,
		queryWordsAcrossReplicasFrom(
			queryWords,
			discovery.askRun.askOutcomes,
			discovery.partitions,
		),
		discovery.partitions,
	)
}

func (discovery *discovery) askTheRestAfter(
	leadingQueryWordFromTheSample yacymodel.Optional[yacymodel.Hash],
) {
	leadingQueryWord, chosen := leadingQueryWordFromTheSample.Get()
	if !chosen {
		discovery.askRun.put(discovery.asks)

		return
	}
	discovery.askAroundTheLeadingWord(queryWordRolesAround(leadingQueryWord, discovery.query))
}

func (discovery *discovery) askAroundTheLeadingWord(roles queryWordRoles) {
	discovery.askRun.put(discovery.asks.ofWords(roles.wordsOfTheDocumentsToMatch))
	partitionsLeft := partitionsOfTheRing(discovery.partitions)
	for {
		settledPartitions := discovery.partitionsSettledAmong(
			partitionsLeft, roles.wordsOfTheDocumentsToMatch,
		)
		for _, partition := range settledPartitions {
			discovery.askForTheOtherWordsIn(partition, roles)
		}
		partitionsLeft = partitionsWithout(partitionsLeft, settledPartitions)
		if len(partitionsLeft) == 0 {
			return
		}
		discovery.askRun.readTheNextSettledWordPartition()
	}
}

func partitionsOfTheRing(partitions yacymodel.DHTRingPartitions) []uint {
	partitionsOfTheRing := make([]uint, 0, partitions)
	for partition := range uint(partitions) {
		partitionsOfTheRing = append(partitionsOfTheRing, partition)
	}

	return partitionsOfTheRing
}

func (discovery *discovery) partitionsSettledAmong(
	partitions []uint,
	words []yacymodel.Hash,
) []uint {
	var settledPartitions []uint
	for _, partition := range partitions {
		if discovery.askRun.haveSettled(discovery.asks.ofWordsIn(words, partition)) {
			settledPartitions = append(settledPartitions, partition)
		}
	}

	return settledPartitions
}

func partitionsWithout(partitions []uint, partitionsLeftOut []uint) []uint {
	return slices.DeleteFunc(partitions, func(partition uint) bool {
		return slices.Contains(partitionsLeftOut, partition)
	})
}

func (discovery *discovery) askForTheOtherWordsIn(partition uint, roles queryWordRoles) {
	asksOfTheOtherWords := discovery.askRun.notPutAmong(
		discovery.asks.ofWordsIn(roles.otherWords, partition),
	)
	if len(asksOfTheOtherWords) == 0 {
		return
	}
	documentsToMatch := discovery.documentsToMatchIn(partition, roles.wordsOfTheDocumentsToMatch)
	otherWordAsks := otherWordAsksFrom(documentsToMatch, discovery.documentsToMatchCeiling)
	discovery.askRun.put(otherWordAsks.appliedTo(asksOfTheOtherWords, documentsToMatch))
	discovery.otherWordAsksPerPartition[partition] = otherWordAsks
}

func (discovery *discovery) documentsToMatchIn(
	partition uint,
	wordsOfTheDocumentsToMatch []yacymodel.Hash,
) []yacymodel.URLHash {
	answeredAsks := discovery.askRun.askOutcomes.AnsweredAsks()
	documentsToMatch := distinctDocuments{}
	for _, answeredAsk := range answeredAsks {
		if !slices.Contains(wordsOfTheDocumentsToMatch, answeredAsk.Ask.Word) {
			continue
		}
		for _, document := range answeredAsk.Abstract {
			if discovery.partitions.PartitionOf(document) != partition {
				continue
			}
			documentsToMatch.add(document)
		}
	}

	return holdersPerDocumentOf(answeredAsks).mostHeldFirst(documentsToMatch)
}

func (discovery *discovery) roundFrom(
	sampledPartition uint,
	leadingQueryWordFromTheSample yacymodel.Optional[yacymodel.Hash],
) discoveryRound {
	askOutcomes := discovery.askRun.askOutcomes
	answeredAsks := askOutcomes.AnsweredAsks()
	queryWordsFewestDocumentsFirst := queryWordsFewestDocumentsFirstFrom(
		discovery.query.WordHashes(), askOutcomes, discovery.partitions,
	)

	return discoveryRound{
		queryWords:                     discovery.query.WordHashes(),
		answeredAsks:                   answeredAsks,
		queryWordsFewestDocumentsFirst: queryWordsFewestDocumentsFirst,
		compoundWords: compoundWordsAcrossReplicasFrom(
			discovery.query.CompoundWords, askOutcomes, discovery.partitions,
		),
		holdersPerDocument: holdersPerDocumentOf(answeredAsks),
		sampledPartition:   sampledPartition,
		amountOfQueryWordsWithASample: amountOfQueryWordsWithASampleIn(
			sampledPartition, queryWordsFewestDocumentsFirst, discovery.partitions,
		),
		leadingQueryWordFromTheSample: leadingQueryWordFromTheSample,
		otherWordAsksPerPartition:     discovery.otherWordAsksPerPartition,
	}
}
