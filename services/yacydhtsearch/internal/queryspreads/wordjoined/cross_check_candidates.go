package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type crossCheckCandidates struct {
	ofWordPartitionsWithoutACompleteAbstract []crossCheckCandidatesOfWordPartition
	ofWordPartitionsWithACompleteAbstract    []crossCheckCandidatesOfWordPartition
}

type crossCheckCandidatesOfWordPartition struct {
	wordPartition wordPartition
	documents     []yacymodel.URLHash
}

func crossCheckCandidatesIn(
	discoveryRound discoveryRound,
	partitions yacymodel.DHTRingPartitions,
) crossCheckCandidates {
	var candidates crossCheckCandidates
	for _, candidatesOfWordPartition := range crossCheckCandidatesPerWordPartitionIn(
		discoveryRound, partitions,
	) {
		if candidatesOfWordPartition.wordPartition.hasACompleteAbstract() {
			candidates.ofWordPartitionsWithACompleteAbstract = append(
				candidates.ofWordPartitionsWithACompleteAbstract, candidatesOfWordPartition,
			)

			continue
		}
		candidates.ofWordPartitionsWithoutACompleteAbstract = append(
			candidates.ofWordPartitionsWithoutACompleteAbstract, candidatesOfWordPartition,
		)
	}

	return candidates
}

func crossCheckCandidatesPerWordPartitionIn(
	discoveryRound discoveryRound,
	partitions yacymodel.DHTRingPartitions,
) []crossCheckCandidatesOfWordPartition {
	documentsOfTheLeadingQueryWord := discoveryRound.holdersPerDocument.mostHeldFirst(
		discoveryRound.leadingQueryWord().documents(),
	)
	var candidates []crossCheckCandidatesOfWordPartition
	for _, queryWord := range discoveryRound.queryWordsBesideTheLeadingQueryWord() {
		documentsPerPartition := documentsPerPartitionFrom(
			queryWord.documentsOutsideAmong(documentsOfTheLeadingQueryWord),
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

func (candidatesOfWordPartition crossCheckCandidatesOfWordPartition) mostHeldDocumentsUpTo(
	documentsToMatchCeiling int,
) []yacymodel.URLHash {
	return candidatesOfWordPartition.documents[:min(
		len(candidatesOfWordPartition.documents), documentsToMatchCeiling,
	)]
}
