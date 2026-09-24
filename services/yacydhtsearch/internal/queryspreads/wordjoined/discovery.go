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

func (discovery *discovery) roundStartingIn(sampledPartition uint) discoveryRound {
	sample := discovery.askForTheSampleIn(sampledPartition)
	sampledLeadingQueryWord := sample.rarestQueryWord()
	discovery.askTheRestAfter(sampledLeadingQueryWord)
	discovery.askRun.finish()

	return discovery.roundFrom(sample, sampledLeadingQueryWord)
}

func (discovery *discovery) askForTheSampleIn(sampledPartition uint) queryWordSample {
	queryWords := discovery.query.WordHashes()
	asksOfTheSample := discovery.asks.ofWordsIn(queryWords, sampledPartition)
	discovery.askRun.put(asksOfTheSample)
	discovery.askRun.readUntilSettled(asksOfTheSample)

	return queryWordSampleIn(
		sampledPartition, queryWords, discovery.askRun.askOutcomes, discovery.partitions,
	)
}

func (discovery *discovery) askTheRestAfter(
	sampledLeadingQueryWord yacymodel.Optional[yacymodel.Hash],
) {
	leadingQueryWord, sampled := sampledLeadingQueryWord.Get()
	if !sampled {
		discovery.askRun.put(discovery.asks)

		return
	}
	discovery.askAroundTheLeadingWord(queryWordRolesAround(leadingQueryWord, discovery.query))
}

func (discovery *discovery) askAroundTheLeadingWord(roles queryWordRoles) {
	discovery.askRun.put(discovery.asks.ofWords(roles.wordsOfTheDocumentsToMatch))
	discovery.askForTheOtherWordsInNewlySettledPartitions(roles)
	for len(discovery.otherWordAsksPerPartition) < int(discovery.partitions) {
		discovery.askRun.readTheNextSettledWordPartition()
		discovery.askForTheOtherWordsInNewlySettledPartitions(roles)
	}
}

func (discovery *discovery) askForTheOtherWordsInNewlySettledPartitions(roles queryWordRoles) {
	for partition := range uint(discovery.partitions) {
		if _, asked := discovery.otherWordAsksPerPartition[partition]; asked ||
			!discovery.askRun.haveSettled(
				discovery.asks.ofWordsIn(roles.wordsOfTheDocumentsToMatch, partition),
			) {
			continue
		}
		discovery.otherWordAsksPerPartition[partition] = discovery.askForTheOtherWordsIn(
			partition, roles,
		)
	}
}

func (discovery *discovery) askForTheOtherWordsIn(
	partition uint,
	roles queryWordRoles,
) OtherWordAsks {
	documentsToMatch := discovery.documentsToMatchIn(partition, roles.wordsOfTheDocumentsToMatch)
	otherWordAsks := otherWordAsksFrom(documentsToMatch, discovery.documentsToMatchCeiling)
	discovery.askRun.put(otherWordAsks.asksFrom(
		discovery.asks.ofWordsIn(roles.otherWords, partition), documentsToMatch,
	))

	return otherWordAsks
}

func (discovery *discovery) documentsToMatchIn(
	partition uint,
	wordsOfTheDocumentsToMatch []yacymodel.Hash,
) []yacymodel.URLHash {
	answeredAsks := discovery.askRun.askOutcomes.AnsweredAsks()
	documentsOfTheWords := distinctDocuments{}
	for _, answeredAsk := range answeredAsks {
		if !slices.Contains(wordsOfTheDocumentsToMatch, answeredAsk.Ask.Word) {
			continue
		}
		for _, document := range answeredAsk.Abstract {
			documentsOfTheWords.add(document)
		}
	}

	return documentsPerPartitionFrom(
		holdersPerDocumentOf(answeredAsks).mostHeldFirst(documentsOfTheWords),
		discovery.partitions,
	)[partition]
}

func documentsPerPartitionFrom(
	documents []yacymodel.URLHash,
	partitions yacymodel.DHTRingPartitions,
) [][]yacymodel.URLHash {
	documentsPerPartition := make([][]yacymodel.URLHash, partitions)
	for _, document := range documents {
		partition := partitions.PartitionOf(document)
		documentsPerPartition[partition] = append(documentsPerPartition[partition], document)
	}

	return documentsPerPartition
}

func (discovery *discovery) roundFrom(
	sample queryWordSample,
	sampledLeadingQueryWord yacymodel.Optional[yacymodel.Hash],
) discoveryRound {
	askOutcomes := discovery.askRun.askOutcomes
	answeredAsks := askOutcomes.AnsweredAsks()

	return discoveryRound{
		queryWords:   discovery.query.WordHashes(),
		answeredAsks: answeredAsks,
		queryWordsFewestDocumentsFirst: queryWordsFewestDocumentsFirstFrom(
			discovery.query.WordHashes(), askOutcomes, discovery.partitions,
		),
		compoundWords: compoundWordsAcrossReplicasFrom(
			discovery.query.CompoundWords, askOutcomes, discovery.partitions,
		),
		holdersPerDocument:        holdersPerDocumentOf(answeredAsks),
		sample:                    sample,
		sampledLeadingQueryWord:   sampledLeadingQueryWord,
		otherWordAsksPerPartition: discovery.otherWordAsksInPartitionOrder(),
	}
}

func (discovery *discovery) otherWordAsksInPartitionOrder() []OtherWordAsks {
	otherWordAsksInPartitionOrder := make(
		[]OtherWordAsks,
		0,
		len(discovery.otherWordAsksPerPartition),
	)
	for partition := range uint(discovery.partitions) {
		otherWordAsks, asked := discovery.otherWordAsksPerPartition[partition]
		if !asked {
			continue
		}
		otherWordAsksInPartitionOrder = append(otherWordAsksInPartitionOrder, otherWordAsks)
	}

	return otherWordAsksInPartitionOrder
}
