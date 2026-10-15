package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpageversion"
)

type URLPageVersionMetrics struct {
	verdicts *prometheus.CounterVec
}

func NewURLPageVersionMetrics(registry prometheus.Registerer) *URLPageVersionMetrics {
	verdicts := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "yacynode_urlpageversion_postings_total",
			Help: "Peer postings by what the node did with them and why, against the page version it indexed.",
		},
		[]string{"outcome", "reason"},
	)
	registry.MustRegister(verdicts)

	return &URLPageVersionMetrics{verdicts: verdicts}
}

func (m *URLPageVersionMetrics) ObserveVerdict(tx *vault.Txn, verdict urlpageversion.Verdict) {
	tx.RunAfterCommit(func() {
		m.verdicts.WithLabelValues(string(verdict.Outcome()), string(verdict)).Inc()
	})
}
