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
	labels     map[seriesID]labels.Labels
	idOfLabels map[string]seriesID
	idsOfName  map[string][]seriesID
	labelsKey  []byte
	nextID     seriesID
}

func newSeriesIndex(stored []indexedSeries) *seriesIndex {
	index := &seriesIndex{
		labels:     map[seriesID]labels.Labels{},
		idOfLabels: map[string]seriesID{},
		idsOfName:  map[string][]seriesID{},
	}
	for _, series := range stored {
		index.put(series)
		index.nextID = max(index.nextID, series.id+1)
	}
	return index
}

func (i *seriesIndex) put(series indexedSeries) {
	i.labels[series.id] = series.labels
	i.idOfLabels[string(series.labels.Bytes(nil))] = series.id
	name := series.labels.Get(model.MetricNameLabel)
	i.idsOfName[name] = append(i.idsOfName[name], series.id)
}

func (i *seriesIndex) add(seriesLabels labels.Labels) (series indexedSeries, isNew bool) {
	i.labelsKey = seriesLabels.Bytes(i.labelsKey[:0])
	if id, known := i.idOfLabels[string(i.labelsKey)]; known {
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
	for name, ids := range i.idsOfName {
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

func (i *seriesIndex) ids() []seriesID {
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
		delete(i.idOfLabels, string(seriesLabels.Bytes(nil)))
		i.forgetNameOf(seriesLabels.Get(model.MetricNameLabel), id)
	}
}

func (i *seriesIndex) forgetNameOf(name string, id seriesID) {
	i.idsOfName[name] = slices.DeleteFunc(
		i.idsOfName[name],
		func(listed seriesID) bool { return listed == id },
	)
	if len(i.idsOfName[name]) == 0 {
		delete(i.idsOfName, name)
	}
}
