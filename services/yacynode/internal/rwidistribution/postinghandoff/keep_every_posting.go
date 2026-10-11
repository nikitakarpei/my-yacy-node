package postinghandoff

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type KeepEveryPosting struct{}

func (KeepEveryPosting) HandOffIfHeldByCloserPeers(
	context.Context,
	*vault.Txn,
	yacymodel.RWIPosting,
	[]yacymodel.Hash,
) (bool, error) {
	return false, nil
}
