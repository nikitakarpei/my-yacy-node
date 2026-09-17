package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	replicaasksobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasksobservers/prometheus"
)

const queryBudget = 3 * time.Second

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func requirePublished(t *testing.T, body string, published []string) {
	t.Helper()

	for _, sample := range published {
		if !strings.Contains(body, sample) {
			t.Fatalf("metrics do not carry %q:\n%s", sample, body)
		}
	}
}

func TestEveryLabelValueOfTheReplicaAsksStartsAtZero(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	replicaasksobserversprometheus.New(registry, queryBudget)

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_replica_asks_duration_seconds_count{ended_by="coverage"} 0`,
		`yacydhtsearch_replica_asks_duration_seconds_count{ended_by="deadline"} 0`,
		`yacydhtsearch_word_partitions_total{settled_by="first"} 0`,
		`yacydhtsearch_word_partitions_total{settled_by="hedge"} 0`,
		`yacydhtsearch_word_partitions_total{settled_by="after empty answer"} 0`,
		`yacydhtsearch_word_partitions_total{settled_by="after failure"} 0`,
		`yacydhtsearch_word_partitions_total{settled_by="exhausted"} 0`,
		`yacydhtsearch_word_partitions_total{settled_by="deadline"} 0`,
		`yacydhtsearch_replica_asks_total{put_as="first"} 0`,
		`yacydhtsearch_replica_asks_total{put_as="hedge"} 0`,
		`yacydhtsearch_replica_asks_total{put_as="after empty answer"} 0`,
		`yacydhtsearch_replica_asks_total{put_as="after failure"} 0`,
		`yacydhtsearch_word_partition_documents_listed_count 0`,
	})
}

func performedReplicaAsks() replicaasks.PerformedReplicaAsks {
	return replicaasks.PerformedReplicaAsks{
		EndedBy:   replicaasks.EndedByCoverage,
		TimeSpent: 250 * time.Millisecond,
		WordPartitions: []replicaasks.PerformedWordPartition{
			{
				SettledBy:               replicaasks.SettledByFirst,
				AmountOfDocumentsListed: 5,
				Asks: []replicaasks.PerformedReplicaAsk{
					{PutAs: replicaasks.PutAsFirst},
				},
			},
			{
				SettledBy:               replicaasks.SettledByHedge,
				AmountOfDocumentsListed: 0,
				Asks: []replicaasks.PerformedReplicaAsk{
					{PutAs: replicaasks.PutAsFirst},
					{PutAs: replicaasks.PutAsHedge},
				},
			},
			{
				SettledBy:               replicaasks.SettledByAfterAFailure,
				AmountOfDocumentsListed: 2,
				Asks: []replicaasks.PerformedReplicaAsk{
					{PutAs: replicaasks.PutAsFirst},
					{PutAs: replicaasks.PutAsAfterAFailure},
				},
			},
		},
	}
}

func TestEveryWordPartitionIsCountedUnderWhatSettledIt(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := replicaasksobserversprometheus.New(registry, queryBudget)

	metrics.ReplicaAsksPerformed(t.Context(), performedReplicaAsks())

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_word_partitions_total{settled_by="first"} 1`,
		`yacydhtsearch_word_partitions_total{settled_by="hedge"} 1`,
		`yacydhtsearch_word_partitions_total{settled_by="after failure"} 1`,
		`yacydhtsearch_word_partitions_total{settled_by="after empty answer"} 0`,
		`yacydhtsearch_word_partitions_total{settled_by="exhausted"} 0`,
		`yacydhtsearch_word_partitions_total{settled_by="deadline"} 0`,
	})
}

func TestEveryReplicaAskIsCountedUnderWhatPutIt(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := replicaasksobserversprometheus.New(registry, queryBudget)

	metrics.ReplicaAsksPerformed(t.Context(), performedReplicaAsks())

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_replica_asks_total{put_as="first"} 3`,
		`yacydhtsearch_replica_asks_total{put_as="hedge"} 1`,
		`yacydhtsearch_replica_asks_total{put_as="after failure"} 1`,
		`yacydhtsearch_replica_asks_total{put_as="after empty answer"} 0`,
	})
}

func TestTheDocumentsOfEverySettledWordPartitionAreMeasured(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := replicaasksobserversprometheus.New(registry, queryBudget)

	metrics.ReplicaAsksPerformed(t.Context(), performedReplicaAsks())

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_word_partition_documents_listed_count 3`,
		`yacydhtsearch_word_partition_documents_listed_sum 7`,
	})
}

func TestTheTimeTheReplicaAsksSpentIsMeasuredUnderWhatEndedThem(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := replicaasksobserversprometheus.New(registry, queryBudget)

	metrics.ReplicaAsksPerformed(t.Context(), performedReplicaAsks())

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_replica_asks_duration_seconds_count{ended_by="coverage"} 1`,
		`yacydhtsearch_replica_asks_duration_seconds_sum{ended_by="coverage"} 0.25`,
		`yacydhtsearch_replica_asks_duration_seconds_count{ended_by="deadline"} 0`,
	})
}
