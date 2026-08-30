package urlmetastaleness

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const (
	orderBucket     vault.Name = "urlmeta_staleness_order"
	freshnessBucket vault.Name = "urlmeta_staleness_freshness"
)

func registerStalenessRanking(
	v *vault.Vault,
) (*vault.Set[rankedURL], *vault.Collection[yacymodel.URLHash, freshnessRank], error) {
	order, err := v.RegisterSet(orderBucket, orderKeyLayout)
	if err != nil {
		return nil, nil, fmt.Errorf("register staleness order: %w", err)
	}
	freshness, err := v.RegisterCollection(
		freshnessBucket,
		freshnessKeyLayout,
		freshnessRankValueCodec{},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register staleness freshness: %w", err)
	}

	return order, freshness, nil
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

var freshnessKeyLayout = vault.SingleKey(hashkeypart.URLHash).KeyLayout()

// freshnessRankValueCodec stores a rank as the plain text it already is, so the
// vault's byte order stays stalest-first.
type freshnessRankValueCodec struct{}

func (freshnessRankValueCodec) Encode(rank freshnessRank) ([]byte, error) {
	return []byte(rank), nil
}

func (freshnessRankValueCodec) Decode(raw []byte) (freshnessRank, error) {
	return freshnessRank(raw), nil
}
