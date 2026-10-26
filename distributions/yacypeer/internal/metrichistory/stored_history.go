package metrichistory

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
	"github.com/prometheus/prometheus/tsdb/chunks"
	"github.com/prometheus/prometheus/tsdb/encoding"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const (
	seriesBucket vault.Name = "series"
	chunksBucket vault.Name = "chunks"
)

var engineLimits = pebblevault.MachineLimits{
	BlockCacheBytes:       8 << 20,
	MemtableBytes:         4 << 20,
	CompactionConcurrency: 1,
}

var (
	seriesKeys = vault.SingleKey(vault.IntegerKeyPart)
	chunkKeys  = vault.PairKey(vault.IntegerKeyPart, vault.IntegerKeyPart)
)

var errBadChunk = errors.New("stored chunk is shorter than its header")

type chunkKey struct {
	series  seriesID
	maxTime int64
}

type chunkSamples struct {
	data    []byte
	minTime int64
}

type cutChunk struct {
	samples            chunkSamples
	key                chunkKey
	replacedCheckpoint *chunkKey
}

type recordedSamples struct {
	series []indexedSeries
	chunks []cutChunk
}

type storedHistory struct {
	vault  *vault.Vault
	series *vault.Collection[seriesID, labels.Labels]
	chunks *vault.Collection[chunkKey, chunkSamples]
}

func openStoredHistory(directory string, clock vault.Clock) (storedHistory, error) {
	engine, err := pebblevault.OpenEngine(directory, 0, engineLimits, nil, nil)
	if err != nil {
		return storedHistory{}, fmt.Errorf("metric history: %w", err)
	}
	storage, err := vault.New(engine, nil, nil, clock)
	if err != nil {
		return storedHistory{}, errors.Join(err, engine.Close())
	}
	series, err := storage.RegisterCollection(
		seriesBucket,
		seriesKeys.KeyLayoutFor(
			func(id seriesID) int64 { return int64(id) },
			func(id int64) seriesID { return seriesID(id) },
		),
		labelsCodec{},
	)
	if err != nil {
		return storedHistory{}, errors.Join(err, storage.Close())
	}
	stored, err := storage.RegisterCollection(
		chunksBucket,
		chunkKeys.KeyLayoutFor(
			func(key chunkKey) (int64, int64) { return int64(key.series), key.maxTime },
			func(series, maxTime int64) chunkKey {
				return chunkKey{series: seriesID(series), maxTime: maxTime}
			},
		),
		chunkSamplesCodec{},
	)
	if err != nil {
		return storedHistory{}, errors.Join(err, storage.Close())
	}
	return storedHistory{vault: storage, series: series, chunks: stored}, nil
}

func (s storedHistory) storedSeries(ctx context.Context) (*seriesIndex, error) {
	var series []indexedSeries
	err := s.vault.View(ctx, func(tx *vault.Txn) error {
		series = series[:0]
		return s.series.Scan(
			tx,
			vault.EveryKey(),
			func(id seriesID, l labels.Labels) (bool, error) {
				series = append(series, indexedSeries{labels: l, id: id})
				return true, nil
			},
		)
	})
	if err != nil {
		return nil, fmt.Errorf("metric history: %w", err)
	}
	return newSeriesIndex(series), nil
}

func (s storedHistory) store(ctx context.Context, recorded recordedSamples) error {
	if len(recorded.series) == 0 && len(recorded.chunks) == 0 {
		return nil
	}
	return s.vault.Update(ctx, func(tx *vault.Txn) error {
		for _, series := range recorded.series {
			if _, err := s.series.Put(tx, series.id, series.labels); err != nil {
				return err
			}
		}
		for _, cut := range recorded.chunks {
			if _, err := s.chunks.Put(tx, cut.key, cut.samples); err != nil {
				return err
			}
			if cut.replacedCheckpoint == nil {
				continue
			}
			if _, err := s.chunks.Delete(tx, *cut.replacedCheckpoint); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s storedHistory) chunksWithin(
	ctx context.Context,
	series []indexedSeries,
	mint, maxt int64,
) (map[seriesID][]chunks.Meta, error) {
	within := map[seriesID][]chunks.Meta{}
	err := s.vault.View(ctx, func(tx *vault.Txn) error {
		clear(within)
		for _, matched := range series {
			metas, err := s.chunksOf(tx, matched.id, mint, maxt)
			if err != nil {
				return err
			}
			within[matched.id] = metas
		}
		return nil
	})
	return within, err
}

func (s storedHistory) chunksOf(
	tx *vault.Txn,
	id seriesID,
	mint, maxt int64,
) ([]chunks.Meta, error) {
	var metas []chunks.Meta
	err := s.chunks.Scan(
		tx,
		chunkKeys.KeysWithFirstBetweenSeconds(int64(id), mint, math.MaxInt64),
		func(key chunkKey, samples chunkSamples) (bool, error) {
			if samples.minTime > maxt {
				return false, nil
			}
			chunk, err := chunkenc.FromData(chunkenc.EncXOR, samples.data)
			if err != nil {
				return false, fmt.Errorf("decode chunk: %w", err)
			}
			metas = append(metas, chunks.Meta{
				Chunk:   chunk,
				MinTime: samples.minTime,
				MaxTime: key.maxTime,
			})
			return true, nil
		},
	)
	return metas, err
}

func (s storedHistory) forgetChunksBefore(
	ctx context.Context,
	cutoff int64,
	ids []seriesID,
) (emptied []seriesID, err error) {
	err = s.vault.Update(ctx, func(tx *vault.Txn) error {
		emptied = emptied[:0]
		for _, id := range ids {
			if _, err := s.chunks.DeleteWithin(
				tx,
				chunkKeys.KeysWithFirstBetweenSeconds(int64(id), math.MinInt64, cutoff-1),
			); err != nil {
				return err
			}
			holdsChunks, err := s.holdsChunks(tx, id)
			if err != nil {
				return err
			}
			if !holdsChunks {
				emptied = append(emptied, id)
			}
		}
		return nil
	})
	return emptied, err
}

func (s storedHistory) holdsChunks(tx *vault.Txn, id seriesID) (bool, error) {
	holds := false
	err := s.chunks.Scan(
		tx,
		chunkKeys.KeysWithFirst(int64(id)),
		func(chunkKey, chunkSamples) (bool, error) {
			holds = true
			return false, nil
		},
	)
	return holds, err
}

func (s storedHistory) forgetSeries(ctx context.Context, ids []seriesID) error {
	if len(ids) == 0 {
		return nil
	}
	return s.vault.Update(ctx, func(tx *vault.Txn) error {
		for _, id := range ids {
			if _, err := s.series.Delete(tx, id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s storedHistory) close() error {
	return s.vault.Close()
}

type labelsCodec struct{}

func (labelsCodec) Encode(seriesLabels labels.Labels) ([]byte, error) {
	var encoded encoding.Encbuf
	seriesLabels.Range(func(label labels.Label) {
		encoded.PutUvarintStr(label.Name)
		encoded.PutUvarintStr(label.Value)
	})
	return encoded.Get(), nil
}

func (labelsCodec) Decode(encoded []byte) (labels.Labels, error) {
	decoded := encoding.Decbuf{B: encoded}
	builder := labels.NewScratchBuilder(0)
	for decoded.Len() > 0 && decoded.Err() == nil {
		name := decoded.UvarintStr()
		builder.Add(name, decoded.UvarintStr())
	}
	if err := decoded.Err(); err != nil {
		return labels.EmptyLabels(), fmt.Errorf("stored series labels: %w", err)
	}
	return builder.Labels(), nil
}

type chunkSamplesCodec struct{}

func (chunkSamplesCodec) Encode(samples chunkSamples) ([]byte, error) {
	encoded := binary.AppendVarint(nil, samples.minTime)
	return append(encoded, samples.data...), nil
}

func (chunkSamplesCodec) Decode(encoded []byte) (chunkSamples, error) {
	minTime, read := binary.Varint(encoded)
	if read <= 0 {
		return chunkSamples{}, errBadChunk
	}
	return chunkSamples{data: bytes.Clone(encoded[read:]), minTime: minTime}, nil
}
