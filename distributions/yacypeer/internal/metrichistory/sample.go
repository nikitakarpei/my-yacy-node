package metrichistory

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

const instanceLabel = "instance"

type sample struct {
	labels labels.Labels
	value  float64
}

type Source struct {
	Gatherer prometheus.Gatherer
	Instance string
}

func samplesOf(ctx context.Context, sources []Source) []sample {
	var samples []sample
	for _, source := range sources {
		samples = source.appendSamplesTo(ctx, samples)
	}
	return samples
}

func (s Source) appendSamplesTo(ctx context.Context, samples []sample) []sample {
	families, err := s.Gatherer.Gather()
	if err != nil {
		slog.WarnContext(ctx, "metric history gathered part of a registry",
			slog.String("instance", s.Instance),
			slog.Any("error", err),
		)
	}
	extracted, err := expfmt.ExtractSamples(&expfmt.DecodeOptions{}, families...)
	if err != nil {
		slog.WarnContext(ctx, "metric history skipped metrics it cannot read",
			slog.String("instance", s.Instance),
			slog.Any("error", err),
		)
	}
	builder := labels.NewScratchBuilder(0)
	for _, found := range extracted {
		samples = append(samples, sample{
			labels: s.labelsOf(&builder, found.Metric),
			value:  float64(found.Value),
		})
	}
	return samples
}

func (s Source) labelsOf(builder *labels.ScratchBuilder, metric model.Metric) labels.Labels {
	builder.Reset()
	for name, value := range metric {
		if name != instanceLabel {
			builder.Add(string(name), string(value))
		}
	}
	builder.Add(instanceLabel, s.Instance)
	builder.Sort()
	return builder.Labels()
}
