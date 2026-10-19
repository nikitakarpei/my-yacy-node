package undeliveredpostings

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type Posting struct {
	WordPartition yacymodel.WordPartition
	URL           yacymodel.URLHash
	entry         undeliveredEntry
}

func undeliveredPostingFrom(
	entry undeliveredEntry,
	partitions yacymodel.DHTRingPartitions,
) Posting {
	return Posting{
		WordPartition: yacymodel.WordPartitionOfWordAndURL(entry.word, entry.url, partitions),
		URL:           entry.url,
		entry:         entry,
	}
}
