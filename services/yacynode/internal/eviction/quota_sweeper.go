package eviction

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmetastaleness"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

const boundReachedMessage = "storage eviction stopped at its sweep bound"

type quotaSweeper struct {
	vault           *vault.Vault
	postings        rwipostings.PostingPurger
	references      urlreferences.ReferenceQuery
	urls            urlmeta.URLEvictor
	stale           urlmetastaleness.StaleURLSource
	target          float64
	urlsPerBatch    int
	batchesPerSweep int
}

func (s quotaSweeper) Sweep(ctx context.Context) (Result, error) {
	quota := s.vault.QuotaBytes()
	if quota <= 0 || s.urlsPerBatch <= 0 || s.batchesPerSweep <= 0 {
		return Result{}, nil
	}
	highWater := int64(float64(quota) * s.target)

	var total Result
	for range s.batchesPerSweep {
		used, err := s.vault.UsedBytes(ctx)
		if err != nil {
			return total, fmt.Errorf("measure usage: %w", err)
		}
		if used < highWater {
			return total, nil
		}

		batch, err := s.purgeStalest(ctx)
		if err != nil {
			return total, err
		}
		total.URLsDeleted += batch.URLsDeleted
		total.PostingsDeleted += batch.PostingsDeleted
		if batch.URLsDeleted == 0 {
			return total, nil
		}
	}

	slog.WarnContext(ctx, boundReachedMessage,
		slog.Int("urls", total.URLsDeleted),
		slog.Int("postings", total.PostingsDeleted),
	)

	return total, nil
}

func (s quotaSweeper) purgeStalest(ctx context.Context) (Result, error) {
	var result Result
	err := s.vault.Update(ctx, func(tx *vault.Txn) error {
		stalest, err := s.stale.StalestURLs(tx, s.urlsPerBatch)
		if err != nil {
			return fmt.Errorf("select stale urls: %w", err)
		}

		purgedPostings, err := s.purgePostings(tx, stalest)
		if err != nil {
			return err
		}

		urlResult, err := s.urls.Purge(ctx, tx, stalest)
		if err != nil {
			return fmt.Errorf("purge urls: %w", err)
		}
		result = Result{URLsDeleted: urlResult.URLsDeleted, PostingsDeleted: purgedPostings}

		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("purge batch: %w", err)
	}

	return result, nil
}

func (s quotaSweeper) purgePostings(tx *vault.Txn, urls []yacymodel.URLHash) (int, error) {
	purged := 0
	for _, url := range urls {
		words, err := s.references.WordsReferencing(tx, url)
		if err != nil {
			return 0, fmt.Errorf("words referencing url: %w", err)
		}
		for _, word := range words {
			deleted, err := s.postings.PurgePosting(tx, word, url)
			if err != nil {
				return 0, fmt.Errorf("purge posting: %w", err)
			}
			if deleted {
				purged++
			}
		}
	}

	return purged, nil
}
