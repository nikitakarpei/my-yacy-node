package urlmetastaleness

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const (
	orderBucket     vault.Name = "urlmeta_staleness_order"
	freshnessBucket vault.Name = "urlmeta_staleness_freshness"
)

func registerStalenessRanking(
	v *vault.Vault,
) (*vault.Set[rankedURL], *vault.Collection[yacymodel.URLHash, freshnessRank], error) {
	order, err := vault.RegisterSet(v, orderBucket, orderKeyCodec{})
	if err != nil {
		return nil, nil, fmt.Errorf("register staleness order: %w", err)
	}
	freshness, err := vault.RegisterCollection(
		v,
		freshnessBucket,
		freshnessKeyCodec{},
		freshnessRankValueCodec{},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register staleness freshness: %w", err)
	}

	return order, freshness, nil
}

var freshnessRankCodec = vaultkey.TextAs(
	func(rank freshnessRank) string { return string(rank) },
	func(text string) (freshnessRank, error) { return freshnessRank(text), nil },
)

var orderKeyLayout = vaultkey.Pair(freshnessRankCodec, hashcodec.URLHash)

type orderKeyCodec struct{}

func (orderKeyCodec) Encode(ranked rankedURL) vaultkey.Key {
	return orderKeyLayout.Key(ranked.rank, ranked.hash)
}

func (orderKeyCodec) Decode(key vaultkey.Key) (rankedURL, error) {
	rank, hash, err := orderKeyLayout.Parts(key)
	if err != nil {
		return rankedURL{}, fmt.Errorf("staleness order key: %w", err)
	}

	return rankedURL{rank: rank, hash: hash}, nil
}

var freshnessKeyLayout = vaultkey.Single(hashcodec.URLHash)

type freshnessKeyCodec struct{}

func (freshnessKeyCodec) Encode(hash yacymodel.URLHash) vaultkey.Key {
	return freshnessKeyLayout.Key(hash)
}

func (freshnessKeyCodec) Decode(key vaultkey.Key) (yacymodel.URLHash, error) {
	hash, err := freshnessKeyLayout.Parts(key)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("staleness freshness key: %w", err)
	}

	return hash, nil
}

// freshnessRankValueCodec stores a rank as the plain text it already is, so the
// vault's byte order stays stalest-first.
type freshnessRankValueCodec struct{}

func (freshnessRankValueCodec) Encode(rank freshnessRank) ([]byte, error) {
	return []byte(rank), nil
}

func (freshnessRankValueCodec) Decode(raw []byte) (freshnessRank, error) {
	return freshnessRank(raw), nil
}
