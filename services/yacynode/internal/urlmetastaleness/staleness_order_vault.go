package urlmetastaleness

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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
	freshness, err := vault.Register(
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

var orderKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

type orderKeyCodec struct{}

func (orderKeyCodec) Encode(ranked rankedURL) vaultkey.Key {
	return orderKeyLayout.Key(string(ranked.rank), ranked.hash.String())
}

func (orderKeyCodec) Decode(key vaultkey.Key) (rankedURL, error) {
	rank, hash, err := orderKeyLayout.Parts(key)
	if err != nil {
		return rankedURL{}, fmt.Errorf("staleness order key: %w", err)
	}

	parsedHash, err := yacymodel.ParseURLHash(hash)
	if err != nil {
		return rankedURL{}, fmt.Errorf("staleness order url hash: %w", err)
	}

	return rankedURL{rank: freshnessRank(rank), hash: parsedHash}, nil
}

var freshnessKeyLayout = vaultkey.Single(vaultkey.Text)

type freshnessKeyCodec struct{}

func (freshnessKeyCodec) Encode(hash yacymodel.URLHash) vaultkey.Key {
	return freshnessKeyLayout.Key(hash.String())
}

func (freshnessKeyCodec) Decode(key vaultkey.Key) (yacymodel.URLHash, error) {
	hash, err := freshnessKeyLayout.Parts(key)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("staleness freshness key: %w", err)
	}

	return yacymodel.ParseURLHash(hash)
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
