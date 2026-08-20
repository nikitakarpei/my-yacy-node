package urlmetastaleness

import (
	"context"
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

type stalenessRanking struct {
	vault     *vault.Vault
	order     *vault.Set[rankedURL]
	freshness *vault.Collection[yacymodel.URLHash, freshnessRank]
}

func openStalenessRanking(v *vault.Vault) (*stalenessRanking, error) {
	order, freshness, err := registerStalenessRanking(v)
	if err != nil {
		return nil, err
	}

	return &stalenessRanking{vault: v, order: order, freshness: freshness}, nil
}

func (o *stalenessRanking) StalestURLs(
	ctx context.Context,
	limit int,
) ([]yacymodel.URLHash, error) {
	if limit <= 0 {
		return nil, nil
	}

	stalest := make([]yacymodel.URLHash, 0, limit)
	err := o.vault.View(ctx, func(tx *vault.Txn) error {
		return o.order.Scan(
			tx,
			vault.EveryKey(),
			func(ranked rankedURL) (bool, error) {
				stalest = append(stalest, ranked.hash)

				return len(stalest) < limit, nil
			},
		)
	})
	if err != nil {
		return nil, fmt.Errorf("select stalest urls: %w", err)
	}

	return stalest, nil
}

func (o *stalenessRanking) URLStored(
	tx *vault.Txn,
	hash yacymodel.URLHash,
	freshness yacymodel.Optional[yacymodel.CalendarDay],
) error {
	rank := rankOf(freshness)
	if err := o.order.Add(tx, rankedURL{rank: rank, hash: hash}); err != nil {
		return fmt.Errorf("record staleness order: %w", err)
	}
	if err := o.freshness.Put(tx, hash, rank); err != nil {
		return fmt.Errorf("record staleness freshness: %w", err)
	}

	return nil
}

var _ StalenessRanking = (*stalenessRanking)(nil)

func (o *stalenessRanking) URLPurged(tx *vault.Txn, hash yacymodel.URLHash) error {
	rank, found, err := o.freshness.Get(tx, hash)
	if err != nil {
		return fmt.Errorf("read staleness freshness: %w", err)
	}
	if !found {
		return nil
	}
	if _, err := o.order.Remove(tx, rankedURL{rank: rank, hash: hash}); err != nil {
		return fmt.Errorf("drop staleness order: %w", err)
	}
	if _, err := o.freshness.Delete(tx, hash); err != nil {
		return fmt.Errorf("drop staleness freshness: %w", err)
	}

	return nil
}
