package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
)

const remoteIndexAdmissionMetrics = `
# HELP yacynode_remoteindexadmission_received_postings_total Postings sent by peers, by what the node did with them.
# TYPE yacynode_remoteindexadmission_received_postings_total counter
yacynode_remoteindexadmission_received_postings_total{outcome="held_in_escrow"} 2
yacynode_remoteindexadmission_received_postings_total{outcome="refused_storage_full"} 3
# HELP yacynode_remoteindexadmission_released_postings_total Postings released from escrow when their URL metadata arrived, by what the node did with them.
# TYPE yacynode_remoteindexadmission_released_postings_total counter
yacynode_remoteindexadmission_released_postings_total{outcome="indexed"} 1
# HELP yacynode_remoteindexadmission_url_metadata_total URL metadata records sent by peers, by what the node did with them.
# TYPE yacynode_remoteindexadmission_url_metadata_total counter
yacynode_remoteindexadmission_url_metadata_total{outcome="replaced_older_version"} 1
`

func TestRemoteIndexAdmissionCountsEachOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewRemoteIndexAdmissionMetrics(registry)

	observer.ObserveReceivedPostings(remoteindexadmission.PostingHeldInEscrow, 1)
	observer.ObserveReceivedPostings(remoteindexadmission.PostingHeldInEscrow, 1)
	observer.ObserveReceivedPostings(remoteindexadmission.PostingRefusedStorageFull, 3)
	observer.ObserveReleasedPostings(remoteindexadmission.PostingIndexed, 1)
	observer.ObserveURLMetadata(remoteindexadmission.URLMetadataReplacedOlderVersion, 1)

	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(remoteIndexAdmissionMetrics),
		"yacynode_remoteindexadmission_received_postings_total",
		"yacynode_remoteindexadmission_released_postings_total",
		"yacynode_remoteindexadmission_url_metadata_total",
	); err != nil {
		t.Fatal(err)
	}
}
