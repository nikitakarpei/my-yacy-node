// Package prometheus reports how often the earned presence of peers reaches
// its snapshots and how often a snapshot could not be read or written.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	labelFailure               = "failure"
	failureSnapshotsUnread     = "snapshotsUnread"
	failureSnapshotUndecodable = "snapshotUndecodable"
	failureSnapshotUnwritten   = "snapshotUnwritten"
)

type PeerPresenceMetrics struct {
	snapshotFailures *prometheusclient.CounterVec
	snapshotsWritten prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *PeerPresenceMetrics {
	metrics := &PeerPresenceMetrics{
		snapshotFailures: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_peer_presence_snapshot_failures_total",
			Help: "Snapshots of earned presence that failed, by failure.",
		}, []string{labelFailure}),
		snapshotsWritten: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_peer_presence_snapshots_written_total",
			Help: "Snapshots of earned presence written to the bucket.",
		}),
	}
	registry.MustRegister(metrics.snapshotFailures, metrics.snapshotsWritten)

	return metrics
}

func (m *PeerPresenceMetrics) SnapshotsReadFailed(context.Context, error) {
	m.snapshotFailures.WithLabelValues(failureSnapshotsUnread).Inc()
}

func (m *PeerPresenceMetrics) SnapshotUndecodable(context.Context, string, error) {
	m.snapshotFailures.WithLabelValues(failureSnapshotUndecodable).Inc()
}

func (m *PeerPresenceMetrics) SnapshotWriteFailed(context.Context, string, error) {
	m.snapshotFailures.WithLabelValues(failureSnapshotUnwritten).Inc()
}

func (m *PeerPresenceMetrics) PeersSnapshotted(_ context.Context, amountOfSnapshots int) {
	m.snapshotsWritten.Add(float64(amountOfSnapshots))
}
