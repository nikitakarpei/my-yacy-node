package undeliveredpostings

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Posting struct {
	WordPartition yacymodel.WordPartition
	URL           yacymodel.URLHash
	change        vault.Change[undeliveredEntry]
}

func undeliveredPostingFrom(
	change vault.Change[undeliveredEntry],
	partitions yacymodel.DHTRingPartitions,
) Posting {
	return Posting{
		WordPartition: yacymodel.WordPartitionOfWordAndURL(
			change.Entry.word,
			change.Entry.url,
			partitions,
		),
		URL:    change.Entry.url,
		change: change,
	}
}
