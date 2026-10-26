package metrichistory

import (
	"slices"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

type seriesID int64

type indexedSeries struct {
	labels labels.Labels
	id     seriesID
}

type seriesIndex struct {
	labels    map[seriesID]labels.Labels
	ids       map[string]seriesID
	named     map[string][]seriesID
	keyBuffer []byte
	nextID    seriesID
}

func newSeriesIndex(stored []indexedSeries) *seriesIndex {
	index := &seriesIndex{
		labels: map[seriesID]labels.Labels{},
		ids:    map[string]seriesID{},
		named:  map[string][]seriesID{},
	}
	for _, series := range stored {
		index.put(series)
		index.nextID = max(index.nextID, series.id+1)
	}
	return index
}

func (i *seriesIndex) put(series indexedSeries) {
	i.labels[series.id] = series.labels
	i.ids[string(series.labels.Bytes(nil))] = series.id
	name := series.labels.Get(model.MetricNameLabel)
	i.named[name] = append(i.named[name], series.id)
}

func (i *seriesIndex) add(seriesLabels labels.Labels) (series indexedSeries, isNew bool) {
	i.keyBuffer = seriesLabels.Bytes(i.keyBuffer[:0])
	if id, known := i.ids[string(i.keyBuffer)]; known {
		return indexedSeries{labels: seriesLabels, id: id}, false
	}
	series = indexedSeries{labels: seriesLabels, id: i.nextID}
	i.nextID++
	i.put(series)
	return series, true
}

func (i *seriesIndex) matching(matchers []*labels.Matcher) []indexedSeries {
	selector := labels.Selector(matchers)
	var matched []indexedSeries
	for _, id := range i.candidatesFor(matchers) {
		if seriesLabels := i.labels[id]; selector.Matches(seriesLabels) {
			matched = append(matched, indexedSeries{labels: seriesLabels, id: id})
		}
	}
	return matched
}

func (i *seriesIndex) candidatesFor(matchers []*labels.Matcher) []seriesID {
	var candidates []seriesID
	nameMatcher := i.nameMatcherOf(matchers)
	for name, ids := range i.named {
		if nameMatcher == nil || nameMatcher.Matches(name) {
			candidates = append(candidates, ids...)
		}
	}
	return candidates
}

func (*seriesIndex) nameMatcherOf(matchers []*labels.Matcher) *labels.Matcher {
	for _, matcher := range matchers {
		if matcher.Name == model.MetricNameLabel {
			return matcher
		}
	}
	return nil
}

func (i *seriesIndex) seriesIDs() []seriesID {
	ids := make([]seriesID, 0, len(i.labels))
	for id := range i.labels {
		ids = append(ids, id)
	}
	return ids
}

func (i *seriesIndex) forget(ids []seriesID) {
	for _, id := range ids {
		seriesLabels, known := i.labels[id]
		if !known {
			continue
		}
		delete(i.labels, id)
		delete(i.ids, string(seriesLabels.Bytes(nil)))
		i.forgetNameOf(seriesLabels.Get(model.MetricNameLabel), id)
	}
}

func (i *seriesIndex) forgetNameOf(name string, id seriesID) {
	i.named[name] = slices.DeleteFunc(
		i.named[name],
		func(named seriesID) bool { return named == id },
	)
	if len(i.named[name]) == 0 {
		delete(i.named, name)
	}
}
