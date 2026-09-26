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
	rememberedDocumentAmounts map[yacymodel.Hash]int
	partitions                yacymodel.DHTRingPartitions
	documentsToMatchCeiling   int
	otherWordAsksPerPartition map[uint]OtherWordAsks
}

//nolint:revive // argument-limit: the discovery takes its run, asks, query, remembered document amounts, ring and ceiling
func discoveryOver(
	askRun *askRun,
	asks discoveryAsks,
	query searchquery.Query,
	rememberedDocumentAmounts map[yacymodel.Hash]int,
	partitions yacymodel.DHTRingPartitions,
	documentsToMatchCeiling int,
) *discovery {
	return &discovery{
		askRun:                    askRun,
		asks:                      asks,
		query:                     query,
		rememberedDocumentAmounts: rememberedDocumentAmounts,
		partitions:                partitions,
		documentsToMatchCeiling:   documentsToMatchCeiling,
		otherWordAsksPerPartition: map[uint]OtherWordAsks{},
	}
}

func (discovery *discovery) askTheQueryWords(sampledPartition uint) discoveryRound {
	leadingQueryWord := discovery.leadingQueryWordRememberedOrSampledIn(sampledPartition)
	discovery.askTheRestAfter(leadingQueryWord)
	discovery.askRun.finish()

	return discovery.roundFrom(sampledPartition, leadingQueryWord)
}

func (discovery *discovery) leadingQueryWordRememberedOrSampledIn(
	sampledPartition uint,
) chosenLeadingQueryWord {
	rememberedLeadingQueryWord := discovery.rememberedLeadingQueryWord()
	if rememberedLeadingQueryWord.Present() {
		return chosenLeadingQueryWord{
			word:   rememberedLeadingQueryWord,
			choice: RarestQueryWordRemembered,
		}
	}
	sampledLeadingQueryWord := discovery.takeTheSampleIn(sampledPartition)
	if sampledLeadingQueryWord.Present() {
		return chosenLeadingQueryWord{
			word:   sampledLeadingQueryWord,
			choice: RarestQueryWordWithASample,
		}
	}

	return chosenLeadingQueryWord{
		word:   yacymodel.None[yacymodel.Hash](),
		choice: RarestQueryWordWithoutASample,
	}
}

func (discovery *discovery) rememberedLeadingQueryWord() yacymodel.Optional[yacymodel.Hash] {
	return rarestQueryWordAmong(discovery.query.WordHashes(), discovery.rememberedDocumentAmounts)
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
			discovery.askRun.settledAsks,
			discovery.partitions,
		),
		discovery.partitions,
	)
}

func (discovery *discovery) askTheRestAfter(leadingQueryWord chosenLeadingQueryWord) {
	word, chosen := leadingQueryWord.word.Get()
	if !chosen {
		discovery.askRun.put(discovery.asks)

		return
	}
	discovery.askAroundTheLeadingWord(leadingQueryWord, queryWordRolesAround(word, discovery.query))
}

func (discovery *discovery) askAroundTheLeadingWord(
	leadingQueryWord chosenLeadingQueryWord,
	roles queryWordRoles,
) {
	discovery.askRun.put(discovery.asks.ofWords(roles.wordsOfTheDocumentsToMatch))
	if discovery.leadingWordPredictsTheOtherWordsOverTheCeiling(
		leadingQueryWord,
		roles.otherWords,
	) {
		discovery.askTheOtherWordsWholeInEveryPartition(roles.otherWords)

		return
	}
	discovery.askTheOtherWordsAsEachPartitionSettles(roles)
}

func (discovery *discovery) leadingWordPredictsTheOtherWordsOverTheCeiling(
	leadingQueryWord chosenLeadingQueryWord,
	otherWords []yacymodel.Hash,
) bool {
	if leadingQueryWord.choice != RarestQueryWordRemembered || len(otherWords) == 0 {
		return false
	}
	word, _ := leadingQueryWord.word.Get()

	return predictsTheOtherWordsOverTheCeiling(
		discovery.rememberedDocumentAmounts[word],
		discovery.partitions,
		discovery.documentsToMatchCeiling,
	)
}

func (discovery *discovery) askTheOtherWordsWholeInEveryPartition(otherWords []yacymodel.Hash) {
	discovery.askRun.put(discovery.asks.ofWords(otherWords))
	for _, partition := range partitionsOfTheRing(discovery.partitions) {
		discovery.otherWordAsksPerPartition[partition] = OtherWordAsksPredictedOverTheCeiling
	}
}

func partitionsOfTheRing(partitions yacymodel.DHTRingPartitions) []uint {
	partitionsOfTheRing := make([]uint, 0, partitions)
	for partition := range uint(partitions) {
		partitionsOfTheRing = append(partitionsOfTheRing, partition)
	}

	return partitionsOfTheRing
}

func (discovery *discovery) askTheOtherWordsAsEachPartitionSettles(roles queryWordRoles) {
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
		discovery.askRun.readTheNextSettledAsk()
	}
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
	asksOfTheOtherWords := discovery.askRun.notAskedAmong(
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
	documentsToMatch := distinctDocuments{}
	for _, settledAsk := range discovery.askRun.settledAsks {
		if !slices.Contains(wordsOfTheDocumentsToMatch, settledAsk.Word) {
			continue
		}
		for _, answer := range settledAsk.Answers {
			for _, listedDocument := range answer.ListedDocuments {
				if discovery.partitions.PartitionOf(listedDocument.Hash) != partition {
					continue
				}
				documentsToMatch.add(listedDocument.Hash)
			}
		}
	}

	return discovery.askRun.holdersPerDocument.mostHeldFirst(documentsToMatch)
}

func (discovery *discovery) roundFrom(
	sampledPartition uint,
	leadingQueryWord chosenLeadingQueryWord,
) discoveryRound {
	settledAsks := discovery.askRun.settledAsks
	queryWordsFewestDocumentsFirst := queryWordsFewestDocumentsFirstFrom(
		discovery.query.WordHashes(), settledAsks, discovery.partitions,
	)

	return discoveryRound{
		queryWords:                     discovery.query.WordHashes(),
		settledAsks:                    settledAsks,
		queryWordsFewestDocumentsFirst: queryWordsFewestDocumentsFirst,
		compoundWords: compoundWordsAcrossReplicasFrom(
			discovery.query.CompoundWords, settledAsks, discovery.partitions,
		),
		holdersPerDocument: discovery.askRun.holdersPerDocument,
		sampledPartition:   sampledPartition,
		amountOfQueryWordsWithASample: amountOfQueryWordsWithASampleIn(
			sampledPartition, queryWordsFewestDocumentsFirst, discovery.partitions,
		),
		chosenLeadingQueryWord:    leadingQueryWord,
		otherWordAsksPerPartition: discovery.otherWordAsksPerPartition,
	}
}
