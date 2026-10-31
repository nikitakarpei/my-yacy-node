// Package urlmetastaleness orders url metadata from stalest to freshest so
// eviction can name the stalest urls without decompressing a single row. Every
// url whose metadata has no postings root ranks staler than any url whose
// metadata has one. It is a projection of the url metadata this node holds, and vault
// keeps it in step with them.
package urlmetastaleness

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type StaleURLSource interface {
	StalestURLs(tx *vault.Txn, limit int) ([]yacymodel.URLHash, error)
}

func Open(
	v *vault.Vault,
) (StaleURLSource, vault.Projection[yacymodel.URLMetadata], error) {
	order, err := registerStalenessOrder(v)
	if err != nil {
		return nil, nil, err
	}

	return &stalenessRanking{order: order}, order, nil
}
