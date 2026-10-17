package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
)

type RWIAdmissionMetrics struct {
	outcomes *prometheus.CounterVec
}

func NewRWIAdmissionMetrics(registry prometheus.Registerer) *RWIAdmissionMetrics {
	outcomes := counterPerLabelFor(
		"yacynode_rwiadmission_postings_total",
		"Inbound postings by what the node decided for them.",
		"outcome",
	)
	registry.MustRegister(outcomes)

	return &RWIAdmissionMetrics{outcomes: outcomes}
}

func (m *RWIAdmissionMetrics) ObserveOutcome(outcome rwiadmission.Outcome, postings int) {
	m.outcomes.WithLabelValues(string(outcome)).Add(float64(postings))
}
