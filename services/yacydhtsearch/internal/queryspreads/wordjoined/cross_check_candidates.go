package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type crossCheckCandidates struct {
	leftOpenByAPartialListing []crossCheckCandidatesOfWordPartition
	ruledOutByAFullListing    []crossCheckCandidatesOfWordPartition
}

type crossCheckCandidatesOfWordPartition struct {
	wordPartition wordPartition
	documents     []yacymodel.URLHash
}

func crossCheckCandidatesIn(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	partitions yacymodel.DHTRingPartitions,
) crossCheckCandidates {
	var candidates crossCheckCandidates
	for _, candidatesOfWordPartition := range crossCheckCandidatesPerWordPartitionIn(
		matchedAndHeldDocumentsRound, partitions,
	) {
		if candidatesOfWordPartition.wordPartition.isFullyListed() {
			candidates.ruledOutByAFullListing = append(
				candidates.ruledOutByAFullListing, candidatesOfWordPartition,
			)

			continue
		}
		candidates.leftOpenByAPartialListing = append(
			candidates.leftOpenByAPartialListing, candidatesOfWordPartition,
		)
	}

	return candidates
}

func crossCheckCandidatesPerWordPartitionIn(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	partitions yacymodel.DHTRingPartitions,
) []crossCheckCandidatesOfWordPartition {
	documentsOfTheLeadingQueryWord := matchedAndHeldDocumentsRound.
		documentsOfTheLeadingQueryWordMostListedFirst()
	var candidates []crossCheckCandidatesOfWordPartition
	for _, queryWord := range matchedAndHeldDocumentsRound.queryWordsBesideTheLeadingQueryWord() {
		documentsPerPartition := documentsPerPartitionFrom(
			queryWord.documentsNotListedByItsPeersAmong(documentsOfTheLeadingQueryWord),
			partitions,
		)
		for partition, wordPartition := range queryWord.wordPartitions() {
			if len(documentsPerPartition[partition]) == 0 {
				continue
			}
			candidates = append(candidates, crossCheckCandidatesOfWordPartition{
				wordPartition: wordPartition,
				documents:     documentsPerPartition[partition],
			})
		}
	}

	return candidates
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
