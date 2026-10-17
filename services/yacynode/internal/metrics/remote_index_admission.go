package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
)

type RemoteIndexAdmissionMetrics struct {
	postings    *prometheus.CounterVec
	urlMetadata *prometheus.CounterVec
}

func NewRemoteIndexAdmissionMetrics(registry prometheus.Registerer) *RemoteIndexAdmissionMetrics {
	postings := counterPerLabelFor(
		"yacynode_remoteindexadmission_postings_total",
		"Inbound postings by what the node decided for them.",
		"outcome",
	)
	urlMetadata := counterPerLabelFor(
		"yacynode_remoteindexadmission_url_metadata_total",
		"URL metadata records sent by peers, by what the node decided for them.",
		"outcome",
	)
	registry.MustRegister(postings, urlMetadata)

	return &RemoteIndexAdmissionMetrics{postings: postings, urlMetadata: urlMetadata}
}

func (m *RemoteIndexAdmissionMetrics) ObservePostings(
	outcome remoteindexadmission.PostingOutcome,
	postings int,
) {
	m.postings.WithLabelValues(string(outcome)).Add(float64(postings))
}

func (m *RemoteIndexAdmissionMetrics) ObserveURLMetadata(
	outcome remoteindexadmission.URLMetadataOutcome,
	records int,
) {
	m.urlMetadata.WithLabelValues(string(outcome)).Add(float64(records))
}

var _ remoteindexadmission.Observer = (*RemoteIndexAdmissionMetrics)(nil)
