package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type discovery struct {
	run                          replicaasks.Run
	asks                         discoveryAsks
	query                        searchquery.Query
	partitions                   yacymodel.DHTRingPartitions
	documentsToMatchCeiling      int
	askOutcomes                  peerasks.SearchDocumentsAskOutcomes
	settledWordPartitions        map[wordPartitionKey]struct{}
	otherWordsAskingPerPartition map[uint]OtherWordsAsking
}

type wordPartitionKey struct {
	word      yacymodel.Hash
	partition uint
}

func discoveryOver(
	run replicaasks.Run,
	asks discoveryAsks,
	query searchquery.Query,
	partitions yacymodel.DHTRingPartitions,
	documentsToMatchCeiling int,
) *discovery {
	return &discovery{
		run:                          run,
		asks:                         asks,
		query:                        query,
		partitions:                   partitions,
		documentsToMatchCeiling:      documentsToMatchCeiling,
		settledWordPartitions:        map[wordPartitionKey]struct{}{},
		otherWordsAskingPerPartition: map[uint]OtherWordsAsking{},
	}
}

func (discovery *discovery) askToDiscoverSampling(sampledPartition uint) discoveryRound {
	samples := discovery.askForTheSamplesIn(sampledPartition)
	discovery.askTheOtherWordPartitionsAfter(samples)
	discovery.settleEveryWordPartition()

	return discovery.roundOf(samples)
}

func (discovery *discovery) askForTheSamplesIn(sampledPartition uint) queryWordSamples {
	queryWords := discovery.query.WordHashes()
	discovery.send(discovery.asks.ofWordsIn(queryWords, sampledPartition))
	discovery.readUntilSettled(queryWords, sampledPartition)

	return queryWordSamplesIn(
		sampledPartition, queryWords, discovery.askOutcomes, discovery.partitions,
	)
}

func (discovery *discovery) send(asks discoveryAsks) {
	if len(asks) == 0 {
		return
	}
	discovery.run.Asks <- asks
}

func (discovery *discovery) readUntilSettled(words []yacymodel.Hash, partition uint) {
	for !discovery.haveSettled(words, partition) {
		discovery.readTheNextSettledWordPartition()
	}
}

func (discovery *discovery) haveSettled(words []yacymodel.Hash, partition uint) bool {
	for _, ask := range discovery.asks.ofWordsIn(words, partition) {
		if _, settled := discovery.settledWordPartitions[wordPartitionKeyOf(ask)]; !settled {
			return false
		}
	}

	return true
}

func wordPartitionKeyOf(ask peerasks.SearchDocumentsAsk) wordPartitionKey {
	return wordPartitionKey{word: ask.Word, partition: ask.Partition}
}

func (discovery *discovery) readTheNextSettledWordPartition() {
	discovery.record(<-discovery.run.SettledWordPartitions)
}

func (discovery *discovery) record(settledWordPartition replicaasks.SettledWordPartition) {
	discovery.askOutcomes = append(discovery.askOutcomes, settledWordPartition.AskOutcomes...)
	for _, askOutcome := range settledWordPartition.AskOutcomes {
		discovery.settledWordPartitions[wordPartitionKeyOf(askOutcome.Ask)] = struct{}{}
	}
}

func (discovery *discovery) askTheOtherWordPartitionsAfter(samples queryWordSamples) {
	leadingQueryWord, sampled := samples.rarestQueryWord().Get()
	if !sampled {
		discovery.askEveryWordPartitionWithoutASample()

		return
	}
	discovery.askTheOtherWordsAsTheLeadSettles(wordSplitBy(leadingQueryWord, discovery.query))
}

func (discovery *discovery) askEveryWordPartitionWithoutASample() {
	discovery.send(discovery.asks)
	for partition := range uint(discovery.partitions) {
		discovery.otherWordsAskingPerPartition[partition] = OtherWordsAskedWithoutASample
	}
}

func (discovery *discovery) askTheOtherWordsAsTheLeadSettles(split wordSplit) {
	discovery.send(discovery.asks.ofWords(split.candidateWords))
	discovery.askTheOtherWordsWhereTheLeadSettled(split)
	for len(discovery.otherWordsAskingPerPartition) < int(discovery.partitions) {
		discovery.readTheNextSettledWordPartition()
		discovery.askTheOtherWordsWhereTheLeadSettled(split)
	}
}

func (discovery *discovery) askTheOtherWordsWhereTheLeadSettled(split wordSplit) {
	for partition := range uint(discovery.partitions) {
		if _, asked := discovery.otherWordsAskingPerPartition[partition]; asked ||
			!discovery.haveSettled(split.candidateWords, partition) {
			continue
		}
		discovery.otherWordsAskingPerPartition[partition] = discovery.askTheOtherWordsIn(
			partition, split,
		)
	}
}

func (discovery *discovery) askTheOtherWordsIn(partition uint, split wordSplit) OtherWordsAsking {
	candidates := discovery.candidatesIn(partition, split.candidateWords)
	otherWordsAsking := otherWordsAskingFor(
		discovery.wordPartitionOf(split.leadingQueryWord, partition),
		candidates,
		discovery.documentsToMatchCeiling,
	)
	discovery.send(otherWordsAsking.asksAmong(
		discovery.asks.ofWordsIn(split.otherWords, partition), candidates,
	))

	return otherWordsAsking
}

func (discovery *discovery) candidatesIn(
	partition uint,
	candidateWords []yacymodel.Hash,
) []yacymodel.URLHash {
	answeredAsks := discovery.askOutcomes.AnsweredAsks()
	documentsOfTheCandidateWords := distinctDocuments{}
	for _, answeredAsk := range answeredAsks {
		if !slices.Contains(candidateWords, answeredAsk.Ask.Word) {
			continue
		}
		for _, document := range answeredAsk.Abstract {
			documentsOfTheCandidateWords.add(document)
		}
	}

	return documentsPerPartitionFrom(
		holdersPerDocumentOf(answeredAsks).mostHeldFirst(documentsOfTheCandidateWords),
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

func (discovery *discovery) wordPartitionOf(word yacymodel.Hash, partition uint) wordPartition {
	return queryWordAcrossReplicasFrom(word, discovery.askOutcomes, discovery.partitions).
		wordPartitions()[partition]
}

func (discovery *discovery) settleEveryWordPartition() {
	close(discovery.run.Asks)
	for settledWordPartition := range discovery.run.SettledWordPartitions {
		discovery.record(settledWordPartition)
	}
}

func (discovery *discovery) roundOf(samples queryWordSamples) discoveryRound {
	askOutcomes := discovery.askOutcomes
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
		holdersPerDocument:           holdersPerDocumentOf(answeredAsks),
		samples:                      samples,
		otherWordsAskingPerPartition: discovery.otherWordsAskingInPartitionOrder(),
	}
}

func (discovery *discovery) otherWordsAskingInPartitionOrder() []OtherWordsAsking {
	otherWordsAskingInPartitionOrder := make([]OtherWordsAsking, 0, discovery.partitions)
	for partition := range uint(discovery.partitions) {
		otherWordsAskingInPartitionOrder = append(
			otherWordsAskingInPartitionOrder,
			discovery.otherWordsAskingPerPartition[partition],
		)
	}

	return otherWordsAskingInPartitionOrder
}
