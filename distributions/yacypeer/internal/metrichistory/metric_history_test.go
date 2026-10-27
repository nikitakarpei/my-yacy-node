package metrichistory_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb/chunkenc"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/metrichistory"
)

const (
	sampleInterval  = metrichistory.SampleInterval
	instance        = "yacynode"
	retentionInTest = time.Hour
)

var historyStart = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

type steppedClock struct {
	now    time.Time
	asleep chan struct{}
	wake   chan struct{}
	mu     sync.Mutex
}

func newSteppedClock() *steppedClock {
	return &steppedClock{now: historyStart, asleep: make(chan struct{}), wake: make(chan struct{})}
}

func (c *steppedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *steppedClock) Sleep(ctx context.Context, d time.Duration) error {
	select {
	case c.asleep <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("sleep: %w", ctx.Err())
	}
	select {
	case <-c.wake:
	case <-ctx.Done():
		return fmt.Errorf("sleep: %w", ctx.Err())
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	return nil
}

type runningHistory struct {
	history *metrichistory.History
	clock   *steppedClock
	stop    func()
}

func serviceSource(registry prometheus.Gatherer) metrichistory.Source {
	return metrichistory.Source{Gatherer: registry, Instance: instance}
}

func startHistory(
	t *testing.T,
	directory string,
	clock *steppedClock,
	sources ...metrichistory.Source,
) runningHistory {
	t.Helper()
	history, err := metrichistory.Open(context.Background(), directory, retentionInTest, clock)
	if err != nil {
		t.Fatalf("open history: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- history.Run(ctx, sources)
	}()
	<-clock.asleep
	return runningHistory{
		history: history,
		clock:   clock,
		stop: func() {
			cancel()
			if err := <-done; err != nil {
				t.Errorf("run: %v", err)
			}
			if err := history.Close(); err != nil {
				t.Errorf("close: %v", err)
			}
		},
	}
}

func (r runningHistory) sampleAgain(beforeSample func()) {
	beforeSample()
	r.clock.wake <- struct{}{}
	<-r.clock.asleep
}

type recordedSeries struct {
	labels  labels.Labels
	times   []int64
	samples []float64
}

func seriesOf(
	t *testing.T,
	queryable storage.Queryable,
	matchers ...*labels.Matcher,
) []recordedSeries {
	t.Helper()
	querier, err := queryable.Querier(math.MinInt64, math.MaxInt64)
	if err != nil {
		t.Fatalf("querier: %v", err)
	}
	set := querier.Select(context.Background(), true, nil, matchers...)
	var found []recordedSeries
	for set.Next() {
		series := recordedSeries{labels: set.At().Labels()}
		iterator := set.At().Iterator(nil)
		for iterator.Next() == chunkenc.ValFloat {
			at, value := iterator.At()
			series.times = append(series.times, at)
			series.samples = append(series.samples, value)
		}
		found = append(found, series)
	}
	if err := set.Err(); err != nil {
		t.Fatalf("select: %v", err)
	}
	return found
}

func gaugeIn(registry *prometheus.Registry, name string) prometheus.Gauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: name, Help: name})
	registry.MustRegister(gauge)
	return gauge
}

func nameIs(name string) *labels.Matcher {
	return labels.MustNewMatcher(labels.MatchEqual, model.MetricNameLabel, name)
}

func TestHistoryAnswersWithEverySampleOfASeries(t *testing.T) {
	registry := prometheus.NewRegistry()
	gauge := gaugeIn(registry, "known_peers")
	running := startHistory(t, t.TempDir(), newSteppedClock(), serviceSource(registry))
	defer running.stop()

	for value := 1; value < 5; value++ {
		running.sampleAgain(func() { gauge.Set(float64(value)) })
	}

	found := seriesOf(t, running.history, nameIs("known_peers"))
	if len(found) != 1 {
		t.Fatalf("series = %d, want 1", len(found))
	}
	want := labels.FromStrings(model.MetricNameLabel, "known_peers", "instance", instance)
	if !labels.Equal(found[0].labels, want) {
		t.Errorf("labels = %s, want %s", found[0].labels, want)
	}
	if !slices.Equal(found[0].samples, []float64{0, 1, 2, 3, 4}) {
		t.Errorf("samples = %v, want 0 to 4", found[0].samples)
	}
	if step := found[0].times[1] - found[0].times[0]; step != sampleInterval.Milliseconds() {
		t.Errorf("samples are %d ms apart, want %d", step, sampleInterval.Milliseconds())
	}
}

func TestHistoryKeepsItsSamplesAcrossARestart(t *testing.T) {
	directory := t.TempDir()
	registry := prometheus.NewRegistry()
	gauge := gaugeIn(registry, "known_peers")
	running := startHistory(t, directory, newSteppedClock(), serviceSource(registry))
	samples := 130
	for value := 1; value < samples; value++ {
		running.sampleAgain(func() { gauge.Set(float64(value)) })
	}
	running.stop()

	clock := newSteppedClock()
	clock.now = historyStart.Add(time.Duration(samples) * sampleInterval)
	restartedRegistry := prometheus.NewRegistry()
	gaugeIn(restartedRegistry, "crawled_pages").Set(7)
	restarted := startHistory(t, directory, clock, serviceSource(restartedRegistry))
	defer restarted.stop()

	found := seriesOf(t, restarted.history, nameIs("known_peers"))
	if len(found) != 1 || len(found[0].samples) != samples {
		t.Fatalf("found %v, want one series of %d samples", found, samples)
	}
	if last := found[0].samples[samples-1]; last != float64(samples-1) {
		t.Errorf("last sample = %v, want %d", last, samples-1)
	}
	added := seriesOf(t, restarted.history, nameIs("crawled_pages"))
	if len(added) != 1 || !slices.Equal(added[0].samples, []float64{7}) {
		t.Errorf("series added after the restart = %v, want one sample of 7", added)
	}
}

func TestHistoryKeepsCheckpointedSamplesAcrossACrash(t *testing.T) {
	directory := t.TempDir()
	registry := prometheus.NewRegistry()
	gauge := gaugeIn(registry, "known_peers")
	running := startHistory(t, directory, newSteppedClock(), serviceSource(registry))
	samples := 40
	for value := 1; value < samples; value++ {
		running.sampleAgain(func() { gauge.Set(float64(value)) })
	}
	crashed := t.TempDir()
	if err := os.CopyFS(crashed, os.DirFS(directory)); err != nil {
		t.Fatalf("copy the history: %v", err)
	}
	running.stop()

	clock := newSteppedClock()
	clock.now = historyStart.Add(time.Duration(samples) * sampleInterval)
	restarted := startHistory(t, crashed, clock, serviceSource(prometheus.NewRegistry()))
	defer restarted.stop()

	found := seriesOf(t, restarted.history, nameIs("known_peers"))
	if len(found) != 1 || len(found[0].samples) == 0 || len(found[0].samples) >= samples {
		t.Fatalf("found %v, want one series with the samples up to the last checkpoint", found)
	}
	for at, value := range found[0].samples {
		if value != float64(at) {
			t.Fatalf("samples = %v, want the first samples in order", found[0].samples)
		}
	}
}

func TestHistoryForgetsSamplesOlderThanTheRetention(t *testing.T) {
	registry := prometheus.NewRegistry()
	gauge := gaugeIn(registry, "known_peers")
	vanishing := gaugeIn(registry, "vanishing")
	running := startHistory(t, t.TempDir(), newSteppedClock(), serviceSource(registry))
	defer running.stop()

	running.sampleAgain(func() { registry.Unregister(vanishing) })
	samplesOverThreeRetentions := int(3 * retentionInTest / sampleInterval)
	for value := range samplesOverThreeRetentions {
		running.sampleAgain(func() { gauge.Set(float64(value)) })
	}

	found := seriesOf(t, running.history, nameIs("known_peers"))
	if len(found) != 1 {
		t.Fatalf("series = %d, want 1", len(found))
	}
	oldest := time.UnixMilli(found[0].times[0])
	if age := running.clock.Now().Sub(oldest); age > retentionInTest+time.Hour {
		t.Errorf("oldest sample is %s old, want under the retention and a forget interval", age)
	}
	if gone := seriesOf(t, running.history, nameIs("vanishing")); len(gone) != 0 {
		t.Errorf("vanished series = %v, want none", gone)
	}
}

func TestHistoryNamesHistogramAndSummarySeriesAsPrometheusDoes(t *testing.T) {
	registry := prometheus.NewRegistry()
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "answer_seconds", Help: "answer", Buckets: []float64{0.5, 1},
	})
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "wait_seconds", Help: "wait", Objectives: map[float64]float64{0.5: 0.05},
	})
	registry.MustRegister(histogram, summary)
	histogram.Observe(0.7)
	summary.Observe(2)
	running := startHistory(t, t.TempDir(), newSteppedClock(), serviceSource(registry))
	defer running.stop()

	found := seriesOf(t, running.history, labels.MustNewMatcher(
		labels.MatchRegexp, model.MetricNameLabel, "answer_seconds.*|wait_seconds.*",
	))
	want := map[string]float64{
		`{__name__="answer_seconds_bucket", instance="yacynode", le="+Inf"}`: 1,
		`{__name__="answer_seconds_bucket", instance="yacynode", le="0.5"}`:  0,
		`{__name__="answer_seconds_bucket", instance="yacynode", le="1"}`:    1,
		`{__name__="answer_seconds_count", instance="yacynode"}`:             1,
		`{__name__="answer_seconds_sum", instance="yacynode"}`:               0.7,
		`{__name__="wait_seconds", instance="yacynode", quantile="0.5"}`:     2,
		`{__name__="wait_seconds_count", instance="yacynode"}`:               1,
		`{__name__="wait_seconds_sum", instance="yacynode"}`:                 2,
	}
	got := map[string]float64{}
	for _, series := range found {
		got[series.labels.String()] = series.samples[0]
	}
	if len(got) != len(want) {
		t.Errorf("series = %v, want %v", got, want)
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("%s = %v, want %v", name, got[name], value)
		}
	}
}

func TestHistoryListsTheInstancesItHolds(t *testing.T) {
	registry := prometheus.NewRegistry()
	gaugeIn(registry, "known_peers")
	running := startHistory(t, t.TempDir(), newSteppedClock(), serviceSource(registry))
	defer running.stop()

	querier, err := running.history.Querier(math.MinInt64, math.MaxInt64)
	if err != nil {
		t.Fatalf("querier: %v", err)
	}
	values, _, err := querier.LabelValues(
		context.Background(),
		"instance",
		nil,
		nameIs("known_peers"),
	)
	if err != nil {
		t.Fatalf("label values: %v", err)
	}
	if !slices.Equal(values, []string{instance}) {
		t.Errorf("instances = %v, want %s", values, instance)
	}
}

func TestHistoryAnswersNothingOnceClosed(t *testing.T) {
	history, err := metrichistory.Open(
		context.Background(), t.TempDir(), retentionInTest, newSteppedClock(),
	)
	if err != nil {
		t.Fatalf("open history: %v", err)
	}
	if err := history.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if found := seriesOf(t, history, nameIs("known_peers")); len(found) != 0 {
		t.Errorf("series = %v, want none", found)
	}
}
