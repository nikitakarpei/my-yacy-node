package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type URLMetadataIntakeMetrics struct {
	outcomes *prometheus.CounterVec
}

func NewURLMetadataIntakeMetrics(registry prometheus.Registerer) *URLMetadataIntakeMetrics {
	outcomes := counterPerLabelFor(
		"yacynode_urlmeta_peer_metadata_total",
		"URL metadata records sent by peers, by what the node decided for them.",
		"outcome",
	)
	registry.MustRegister(outcomes)

	return &URLMetadataIntakeMetrics{outcomes: outcomes}
}

func (m *URLMetadataIntakeMetrics) ObserveOutcome(outcome urlmeta.Outcome, records int) {
	m.outcomes.WithLabelValues(string(outcome)).Add(float64(records))
}

var _ urlmeta.Observer = (*URLMetadataIntakeMetrics)(nil)
