package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
)

type RWIAdmissionMetrics struct {
	postingsRefused *prometheus.CounterVec
	outcomes        *prometheus.CounterVec
}

func NewRWIAdmissionMetrics(registry prometheus.Registerer) *RWIAdmissionMetrics {
	postingsRefused := counterPerLabelFor(
		"yacynode_rwiadmission_postings_refused_total",
		"Inbound postings the node refused, by the reason it refused them.",
		"reason",
	)
	outcomes := counterPerLabelFor(
		"yacynode_rwiadmission_postings_total",
		"Inbound postings by what happened to them.",
		"outcome",
	)
	registry.MustRegister(postingsRefused, outcomes)

	return &RWIAdmissionMetrics{postingsRefused: postingsRefused, outcomes: outcomes}
}

func (m *RWIAdmissionMetrics) ObserveRefused(reason rwiadmission.RefusalReason, postings int) {
	m.postingsRefused.WithLabelValues(string(reason)).Add(float64(postings))
}

func (m *RWIAdmissionMetrics) ObserveOutcome(tx *vault.Txn, outcome rwiadmission.Outcome) {
	tx.RunAfterCommit(func() { m.outcomes.WithLabelValues(string(outcome)).Inc() })
}
