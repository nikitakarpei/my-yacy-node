package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type crossCheckCandidates struct {
	ofPartlyListedWordPartitions []crossCheckCandidatesOfWordPartition
	ofFullyListedWordPartitions  []crossCheckCandidatesOfWordPartition
}

type crossCheckCandidatesOfWordPartition struct {
	wordPartition wordPartition
	documents     []yacymodel.URLHash
}

func crossCheckCandidatesIn(
	abstractsRound abstractsRound,
	partitions yacymodel.DHTRingPartitions,
) crossCheckCandidates {
	var candidates crossCheckCandidates
	for _, candidatesOfWordPartition := range crossCheckCandidatesPerWordPartitionIn(
		abstractsRound, partitions,
	) {
		if candidatesOfWordPartition.wordPartition.isFullyListed() {
			candidates.ofFullyListedWordPartitions = append(
				candidates.ofFullyListedWordPartitions, candidatesOfWordPartition,
			)

			continue
		}
		candidates.ofPartlyListedWordPartitions = append(
			candidates.ofPartlyListedWordPartitions, candidatesOfWordPartition,
		)
	}

	return candidates
}

func crossCheckCandidatesPerWordPartitionIn(
	abstractsRound abstractsRound,
	partitions yacymodel.DHTRingPartitions,
) []crossCheckCandidatesOfWordPartition {
	documentsOfTheLeadingQueryWord := abstractsRound.
		documentsOfTheLeadingQueryWordMostListedFirst()
	var candidates []crossCheckCandidatesOfWordPartition
	for _, queryWord := range abstractsRound.queryWordsBesideTheLeadingQueryWord() {
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

func (candidatesOfWordPartition crossCheckCandidatesOfWordPartition) mostListedDocumentsUpTo(
	crossCheckedDocumentsCeiling int,
) []yacymodel.URLHash {
	return candidatesOfWordPartition.documents[:min(
		len(candidatesOfWordPartition.documents), crossCheckedDocumentsCeiling,
	)]
}
