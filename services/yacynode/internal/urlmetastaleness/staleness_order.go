package urlmetastaleness

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// freshnessRank orders days as plain text so the vault's byte order is also
// stalest-first order. A url with no known day ranks stalest.
type freshnessRank string

func rankOf(day yacymodel.Optional[yacymodel.CalendarDay]) freshnessRank {
	value, _ := day.Get()

	return freshnessRank(fmt.Sprintf("%04d%02d%02d", value.Year, value.Month, value.Day))
}

type rankedURL struct {
	rank freshnessRank
	hash yacymodel.URLHash
}

func rankedURLOf(metadata yacymodel.URLMetadata) rankedURL {
	return rankedURL{rank: rankOf(metadata.Freshness()), hash: metadata.Hash}
}

type stalenessRanking struct {
	order *vault.ValueIndex[yacymodel.URLMetadata, rankedURL]
}

func (o *stalenessRanking) StalestURLs(tx *vault.Txn, limit int) ([]yacymodel.URLHash, error) {
	if limit <= 0 {
		return nil, nil
	}

	stalest := make([]yacymodel.URLHash, 0, limit)
	if err := o.order.Scan(
		tx,
		vault.EveryKey(),
		func(ranked rankedURL) (bool, error) {
			stalest = append(stalest, ranked.hash)

			return len(stalest) < limit, nil
		},
	); err != nil {
		return nil, fmt.Errorf("select stalest urls: %w", err)
	}

	return stalest, nil
}

var _ StaleURLSource = (*stalenessRanking)(nil)
