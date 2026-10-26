// Package metrichistory samples the metrics of the peer and its services, keeps them on
// disk for a retention period, and lets the PromQL engine read them.
package metrichistory

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb/chunks"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	SampleInterval  = 15 * time.Second
	samplesPerChunk = 120
	idleSeriesAfter = 4 * SampleInterval
)

type History struct {
	clock     vault.Clock
	series    *seriesIndex
	open      *openChunks
	stored    storedHistory
	retention time.Duration
	mu        sync.RWMutex
	closed    bool
}

func Open(
	ctx context.Context,
	directory string,
	retention time.Duration,
	clock vault.Clock,
) (*History, error) {
	stored, err := openStoredHistory(directory, clock)
	if err != nil {
		return nil, err
	}
	series, err := stored.storedSeries(ctx)
	if err != nil {
		return nil, errors.Join(err, stored.close())
	}
	return &History{
		clock:     clock,
		series:    series,
		open:      newOpenChunks(),
		stored:    stored,
		retention: retention,
	}, nil
}

func (h *History) Run(ctx context.Context, sources []Source) error {
	defer h.storeOpenChunks(context.WithoutCancel(ctx))
	for sampleCount := 0; ; sampleCount++ {
		if sampleCount%samplesPerChunk == 0 {
			h.storeIdleChunks(ctx)
			h.forgetExpired(ctx)
		}
		h.record(ctx, samplesOf(ctx, sources))
		if err := h.clock.Sleep(ctx, SampleInterval); err != nil {
			return nil
		}
	}
}

func (h *History) storeOpenChunks(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.stored.store(ctx, recordedSamples{chunks: h.open.cutAll()}); err != nil {
		slog.WarnContext(ctx, "metric history lost its open chunks", slog.Any("error", err))
	}
}

func (h *History) storeIdleChunks(ctx context.Context) {
	idleSince := h.clock.Now().Add(-idleSeriesAfter).UnixMilli()
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.stored.store(
		ctx,
		recordedSamples{chunks: h.open.cutIdleSince(idleSince)},
	); err != nil {
		slog.WarnContext(ctx, "metric history lost idle chunks", slog.Any("error", err))
	}
}

func (h *History) forgetExpired(ctx context.Context) {
	cutoff := h.clock.Now().Add(-h.retention).UnixMilli()
	h.mu.Lock()
	defer h.mu.Unlock()
	emptied, err := h.stored.forgetChunksBefore(ctx, cutoff, h.series.seriesIDs())
	if err != nil {
		slog.WarnContext(ctx, "metric history kept expired chunks", slog.Any("error", err))
		return
	}
	idle := h.open.withoutOpenChunk(emptied)
	if err := h.stored.forgetSeries(ctx, idle); err != nil {
		slog.WarnContext(ctx, "metric history kept expired series", slog.Any("error", err))
		return
	}
	h.series.forget(idle)
	slog.DebugContext(ctx, "metric history forgot expired series", slog.Int("series", len(idle)))
}

func (h *History) record(ctx context.Context, samples []sample) {
	h.mu.Lock()
	defer h.mu.Unlock()
	recorded := h.append(h.clock.Now().UnixMilli(), samples)
	if err := h.stored.store(ctx, recorded); err != nil {
		slog.WarnContext(ctx, "metric history lost full chunks", slog.Any("error", err))
	}
}

func (h *History) append(at int64, samples []sample) (toStore recordedSamples) {
	for _, s := range samples {
		series, isNew := h.series.add(s.labels)
		if isNew {
			toStore.series = append(toStore.series, series)
		}
		if full, isFull := h.open.append(series.id, at, s.value); isFull {
			toStore.chunks = append(toStore.chunks, full)
		}
	}
	return toStore
}

func (h *History) Querier(mint, maxt int64) (storage.Querier, error) {
	return historyQuerier{history: h, mint: mint, maxt: maxt}, nil
}

func (h *History) chunkSeriesWithin(
	ctx context.Context,
	matchers []*labels.Matcher,
	mint, maxt int64,
) ([]storage.ChunkSeries, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.closed {
		return nil, nil
	}
	matched := h.series.matching(matchers)
	stored, err := h.stored.chunksWithin(ctx, matched, mint, maxt)
	if err != nil {
		return nil, err
	}
	selected := make([]storage.ChunkSeries, 0, len(matched))
	for _, series := range matched {
		metas := stored[series.id]
		if open, overlaps := h.open.copyWithin(series.id, mint, maxt); overlaps {
			metas = append(metas, open)
		}
		if len(metas) > 0 {
			selected = append(selected, chunkSeriesOf(series.labels, metas))
		}
	}
	return selected, nil
}

func chunkSeriesOf(seriesLabels labels.Labels, metas []chunks.Meta) storage.ChunkSeries {
	return &storage.ChunkSeriesEntry{
		Lset: seriesLabels,
		ChunkIteratorFn: func(chunks.Iterator) chunks.Iterator {
			return storage.NewListChunkSeriesIterator(metas...)
		},
	}
}

func (h *History) seriesMatching(matchers []*labels.Matcher) []indexedSeries {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.series.matching(matchers)
}

func (h *History) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	return h.stored.close()
}
