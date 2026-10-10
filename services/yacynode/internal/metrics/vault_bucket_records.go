package metrics

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const labelBucket = "bucket"

type VaultBuckets interface {
	RecordCountsByBucket(context.Context) (map[vault.Name]int, error)
}

type VaultBucketRecordMetrics struct {
	buckets      VaultBuckets
	recordCounts *prometheus.Desc
}

func NewVaultBucketRecordMetrics(
	registry prometheus.Registerer,
	buckets VaultBuckets,
) *VaultBucketRecordMetrics {
	metrics := &VaultBucketRecordMetrics{
		buckets: buckets,
		recordCounts: prometheus.NewDesc(
			"yacynode_vault_bucket_records",
			"Records currently stored in each vault bucket of records.",
			[]string{labelBucket},
			nil,
		),
	}
	registry.MustRegister(metrics)

	return metrics
}

func (m *VaultBucketRecordMetrics) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- m.recordCounts
}

func (m *VaultBucketRecordMetrics) Collect(samples chan<- prometheus.Metric) {
	ctx := context.Background()

	recordCounts, err := m.buckets.RecordCountsByBucket(ctx)
	if err != nil {
		slog.WarnContext(ctx, "vault record counts unavailable", slog.Any("error", err))

		return
	}

	for bucket, count := range recordCounts {
		samples <- prometheus.MustNewConstMetric(
			m.recordCounts,
			prometheus.GaugeValue,
			float64(count),
			string(bucket),
		)
	}
}
