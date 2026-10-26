package metrichistory

import (
	"context"
	"log/slog"
	"slices"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/util/annotations"
)

type historyQuerier struct {
	history *History
	mint    int64
	maxt    int64
}

func (q historyQuerier) Select(
	ctx context.Context,
	sortSeries bool,
	_ *storage.SelectHints,
	matchers ...*labels.Matcher,
) storage.SeriesSet {
	selected, err := q.history.chunkSeriesWithin(ctx, matchers, q.mint, q.maxt)
	if err != nil {
		slog.WarnContext(ctx, "metric history could not read chunks", slog.Any("error", err))
		return storage.ErrSeriesSet(err)
	}
	if sortSeries {
		slices.SortFunc(selected, func(a, b storage.ChunkSeries) int {
			return labels.Compare(a.Labels(), b.Labels())
		})
	}
	return storage.NewSeriesSetFromChunkSeriesSet(&chunkSeriesList{series: selected, at: -1})
}

func (q historyQuerier) LabelValues(
	_ context.Context,
	name string,
	_ *storage.LabelHints,
	matchers ...*labels.Matcher,
) ([]string, annotations.Annotations, error) {
	var values []string
	for _, series := range q.history.seriesMatching(matchers) {
		if value := series.labels.Get(name); value != "" {
			values = append(values, value)
		}
	}
	slices.Sort(values)
	return slices.Compact(values), nil, nil
}

func (q historyQuerier) LabelNames(
	_ context.Context,
	_ *storage.LabelHints,
	matchers ...*labels.Matcher,
) ([]string, annotations.Annotations, error) {
	var names []string
	for _, series := range q.history.seriesMatching(matchers) {
		series.labels.Range(func(label labels.Label) {
			names = append(names, label.Name)
		})
	}
	slices.Sort(names)
	return slices.Compact(names), nil, nil
}

func (historyQuerier) Close() error {
	return nil
}

type chunkSeriesList struct {
	series []storage.ChunkSeries
	at     int
}

func (l *chunkSeriesList) Next() bool {
	l.at++
	return l.at < len(l.series)
}

func (l *chunkSeriesList) At() storage.ChunkSeries {
	return l.series[l.at]
}

func (*chunkSeriesList) Err() error {
	return nil
}

func (*chunkSeriesList) Warnings() annotations.Annotations {
	return nil
}
