package documentasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartitionKey struct {
	word      yacymodel.Hash
	partition uint
}

func wordPartitionKeyOf(ask wordpartitionasks.Ask) wordPartitionKey {
	return wordPartitionKey{word: ask.Word, partition: ask.Partition}
}
