package urlmetastaleness

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const orderBucket vault.Name = "urlmeta_staleness_order"

const orderVersion = "1"

func registerStalenessOrder(
	v *vault.Vault,
) (*vault.ValueIndex[yacymodel.URLMetadata, rankedURL], error) {
	order, err := v.RegisterValueIndex(orderBucket, orderVersion, orderKeyLayout, rankedURLOf)
	if err != nil {
		return nil, fmt.Errorf("register staleness order: %w", err)
	}

	return order, nil
}

var freshnessRankKeyPart = vault.TextKeyPartFrom(
	func(rank freshnessRank) string { return string(rank) },
	func(text string) (freshnessRank, error) { return freshnessRank(text), nil },
)

var orderKeyParts = vault.PairKey(freshnessRankKeyPart, hashkeypart.URLHash)

var orderKeyLayout = orderKeyParts.KeyLayoutFor(
	func(ranked rankedURL) (freshnessRank, yacymodel.URLHash) {
		return ranked.rank, ranked.hash
	},
	func(rank freshnessRank, hash yacymodel.URLHash) rankedURL {
		return rankedURL{rank: rank, hash: hash}
	},
)
