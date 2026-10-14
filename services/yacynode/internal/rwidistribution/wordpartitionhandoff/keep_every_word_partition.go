package wordpartitionhandoff

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type KeepEveryWordPartition struct{}

func (KeepEveryWordPartition) HandOffIfHeldByResponsiblePeers(
	context.Context,
	*vault.Txn,
	yacymodel.WordPartition,
	[]yacymodel.Hash,
	[]yacymodel.Hash,
) (int, error) {
	return 0, nil
}
