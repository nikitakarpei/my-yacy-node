package postinghandoff

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type KeepEveryPosting struct{}

func (KeepEveryPosting) HandOffPostingsHeldByCloserPeers(
	context.Context,
	*vault.Txn,
	[]yacymodel.RWIPosting,
) (int, error) {
	return 0, nil
}
