package urlmetastaleness

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const (
	orderBucket     vault.Name = "urlmeta_staleness_order"
	freshnessBucket vault.Name = "urlmeta_staleness_freshness"
)

var freshnessKeyLayout = vaultkey.Single(vaultkey.Text)

type stalenessRanking struct {
	vault     *vault.Vault
	order     *vault.Set
	freshness *vault.Collection[freshnessRank]
}

func openStalenessRanking(v *vault.Vault) (*stalenessRanking, error) {
	order, err := vault.RegisterSet(v, orderBucket)
	if err != nil {
		return nil, fmt.Errorf("register staleness order: %w", err)
	}
	freshness, err := vault.Register(v, freshnessBucket, freshnessRankCodec{})
	if err != nil {
		return nil, fmt.Errorf("register staleness freshness: %w", err)
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
			nil,
			func(key vault.Key) (bool, error) {
				hash, err := hashFromOrderKey(key)
				if err != nil {
					return false, err
				}
				stalest = append(stalest, hash)

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
	if err := o.order.Add(tx, rankedURL{rank: rank, hash: hash}.orderKey()); err != nil {
		return fmt.Errorf("record staleness order: %w", err)
	}
	if err := o.freshness.Put(tx, freshnessKey(hash), rank); err != nil {
		return fmt.Errorf("record staleness freshness: %w", err)
	}

	return nil
}

func freshnessKey(hash yacymodel.URLHash) vault.Key {
	return freshnessKeyLayout.Key(hash.String()).Bytes()
}

var _ StalenessRanking = (*stalenessRanking)(nil)

func (o *stalenessRanking) URLPurged(tx *vault.Txn, hash yacymodel.URLHash) error {
	rank, found, err := o.freshness.Get(tx, freshnessKey(hash))
	if err != nil {
		return fmt.Errorf("read staleness freshness: %w", err)
	}
	if !found {
		return nil
	}
	if _, err := o.order.Remove(tx, rankedURL{rank: rank, hash: hash}.orderKey()); err != nil {
		return fmt.Errorf("drop staleness order: %w", err)
	}
	if _, err := o.freshness.Delete(tx, freshnessKey(hash)); err != nil {
		return fmt.Errorf("drop staleness freshness: %w", err)
	}

	return nil
}
