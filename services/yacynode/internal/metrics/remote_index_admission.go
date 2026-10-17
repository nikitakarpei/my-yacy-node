package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
)

type RemoteIndexAdmissionMetrics struct {
	receivedPostings *prometheus.CounterVec
	releasedPostings *prometheus.CounterVec
	urlMetadata      *prometheus.CounterVec
}

func NewRemoteIndexAdmissionMetrics(registry prometheus.Registerer) *RemoteIndexAdmissionMetrics {
	receivedPostings := counterPerLabelFor(
		"yacynode_remoteindexadmission_received_postings_total",
		"Postings sent by peers, by what the node did with them.",
		"outcome",
	)
	releasedPostings := counterPerLabelFor(
		"yacynode_remoteindexadmission_released_postings_total",
		"Postings released from escrow when their URL metadata arrived, by what the node did with them.",
		"outcome",
	)
	urlMetadata := counterPerLabelFor(
		"yacynode_remoteindexadmission_url_metadata_total",
		"URL metadata records sent by peers, by what the node did with them.",
		"outcome",
	)
	registry.MustRegister(receivedPostings, releasedPostings, urlMetadata)

	return &RemoteIndexAdmissionMetrics{
		receivedPostings: receivedPostings,
		releasedPostings: releasedPostings,
		urlMetadata:      urlMetadata,
	}
}

func (m *RemoteIndexAdmissionMetrics) ObserveReceivedPostings(
	outcome remoteindexadmission.PostingOutcome,
	postings int,
) {
	m.receivedPostings.WithLabelValues(string(outcome)).Add(float64(postings))
}

func (m *RemoteIndexAdmissionMetrics) ObserveReleasedPostings(
	outcome remoteindexadmission.PostingOutcome,
	postings int,
) {
	m.releasedPostings.WithLabelValues(string(outcome)).Add(float64(postings))
}

func (m *RemoteIndexAdmissionMetrics) ObserveURLMetadata(
	outcome remoteindexadmission.URLMetadataOutcome,
	records int,
) {
	m.urlMetadata.WithLabelValues(string(outcome)).Add(float64(records))
}

var _ remoteindexadmission.Observer = (*RemoteIndexAdmissionMetrics)(nil)
