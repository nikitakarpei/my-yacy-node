package vaultmetrics

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

type Capacity interface {
	QuotaBytes() int64
	UsedBytes(context.Context) (int64, error)
}

type CapacityMetrics struct {
	capacity Capacity
	quota    *prometheus.Desc
	used     *prometheus.Desc
}

func NewCapacityMetrics(
	registry prometheus.Registerer,
	serviceMetricNamespace string,
	capacity Capacity,
) *CapacityMetrics {
	metrics := &CapacityMetrics{
		capacity: capacity,
		quota: prometheus.NewDesc(
			serviceMetricNamespace+"_vault_quota_bytes",
			"Configured vault quota in bytes.",
			nil,
			nil,
		),
		used: prometheus.NewDesc(
			serviceMetricNamespace+"_vault_used_bytes",
			"Vault space currently used in bytes.",
			nil,
			nil,
		),
	}
	registry.MustRegister(metrics)

	return metrics
}

func (m *CapacityMetrics) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- m.quota
	descriptions <- m.used
}

func (m *CapacityMetrics) Collect(samples chan<- prometheus.Metric) {
	ctx := context.Background()
	samples <- prometheus.MustNewConstMetric(
		m.quota,
		prometheus.GaugeValue,
		float64(m.capacity.QuotaBytes()),
	)

	used, err := m.capacity.UsedBytes(ctx)
	if err != nil {
		slog.WarnContext(ctx, "vault used bytes unavailable", slog.Any("error", err))

		return
	}

	samples <- prometheus.MustNewConstMetric(m.used, prometheus.GaugeValue, float64(used))
}
