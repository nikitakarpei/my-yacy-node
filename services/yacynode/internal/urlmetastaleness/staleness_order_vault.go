package urlmetastaleness

import (
	"fmt"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const orderBucket vault.Name = "urlmeta_staleness_order"

const orderVersion = "2"

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

var rootedKeyPart = vault.TextKeyPartFrom(strconv.FormatBool, strconv.ParseBool)

var orderKeyParts = vault.TripleKey(rootedKeyPart, freshnessRankKeyPart, hashkeypart.URLHash)

var orderKeyLayout = orderKeyParts.KeyLayoutFor(
	func(ranked rankedURL) (bool, freshnessRank, yacymodel.URLHash) {
		return ranked.rooted, ranked.freshness, ranked.hash
	},
	func(rooted bool, freshness freshnessRank, hash yacymodel.URLHash) rankedURL {
		return rankedURL{rooted: rooted, freshness: freshness, hash: hash}
	},
)
