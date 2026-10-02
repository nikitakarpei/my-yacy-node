package wordpartitionasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Ask struct {
	Word             yacymodel.Hash
	Partition        uint
	HoldersInOrder   []peerdirectory.AskablePeer
	ExcludedWords    []yacymodel.Hash
	Language         string
	DocumentsToMatch []yacymodel.URLHash
}
