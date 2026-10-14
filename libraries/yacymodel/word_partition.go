package yacymodel

type WordPartition struct {
	Word      Hash
	Partition uint
}

func WordPartitionOfPosting(posting RWIPosting, partitions DHTRingPartitions) WordPartition {
	return WordPartitionOfWordAndURL(posting.WordHash, posting.URLHash, partitions)
}

func WordPartitionOfWordAndURL(
	word Hash,
	url URLHash,
	partitions DHTRingPartitions,
) WordPartition {
	return WordPartition{Word: word, Partition: partitions.PartitionOf(url)}
}

func DHTRingPositionOfWordPartition(
	wordPartition WordPartition,
	partitions DHTRingPartitions,
) DHTRingPosition {
	return DHTRingPositionOfWordInPartition(wordPartition.Word, wordPartition.Partition, partitions)
}
