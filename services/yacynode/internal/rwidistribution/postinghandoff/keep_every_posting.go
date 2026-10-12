package postinghandoff

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type KeepEveryPosting struct{}

func (KeepEveryPosting) HandOffIfHeldByResponsiblePeers(
	context.Context,
	*vault.Txn,
	yacymodel.RWIPosting,
	yacymodel.DHTRingPosition,
	[]yacymodel.Hash,
	[]yacymodel.Hash,
) (bool, error) {
	return false, nil
}
