package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpostingpurge"
)

type URLPostingPurgeMetrics struct {
	postings prometheus.Counter
}

func NewURLPostingPurgeMetrics(registry prometheus.Registerer) *URLPostingPurgeMetrics {
	postings := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yacynode_url_posting_purge_postings_total",
		Help: "Postings purged with the url metadata that referenced them.",
	})
	registry.MustRegister(postings)

	return &URLPostingPurgeMetrics{postings: postings}
}

func (m *URLPostingPurgeMetrics) ObservePostingsPurgedWithURL(
	_ yacymodel.URLHash,
	amountOfPostings int,
) {
	m.postings.Add(float64(amountOfPostings))
}

var _ urlpostingpurge.Observer = (*URLPostingPurgeMetrics)(nil)
