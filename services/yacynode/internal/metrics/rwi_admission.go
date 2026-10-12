package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
)

type RWIAdmissionMetrics struct {
	postingsRefused *prometheus.CounterVec
}

func NewRWIAdmissionMetrics(registry prometheus.Registerer) *RWIAdmissionMetrics {
	postingsRefused := counterPerLabelFor(
		"yacynode_rwiadmission_postings_refused_total",
		"Inbound postings the node refused, by the reason it refused them.",
		"reason",
	)
	registry.MustRegister(postingsRefused)

	return &RWIAdmissionMetrics{postingsRefused: postingsRefused}
}

func (m *RWIAdmissionMetrics) ObserveRefused(reason rwiadmission.RefusalReason, postings int) {
	m.postingsRefused.WithLabelValues(string(reason)).Add(float64(postings))
}
