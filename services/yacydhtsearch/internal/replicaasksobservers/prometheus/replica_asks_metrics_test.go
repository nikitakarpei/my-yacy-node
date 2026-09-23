package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
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
		`yacydhtsearch_replica_asks_duration_seconds_count{asked_for="matched and held documents",ended_by="coverage"} 0`,
		`yacydhtsearch_replica_asks_duration_seconds_count{asked_for="matched and held documents",ended_by="deadline"} 0`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="start",settled_by="coverage"} 0`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="hedge delay",settled_by="coverage"} 0`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="empty answer",settled_by="coverage"} 0`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="failure",settled_by="coverage"} 0`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="",settled_by="no replica left"} 0`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="",settled_by="deadline"} 0`,
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="start"} 0`,
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="hedge delay"} 0`,
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="empty answer"} 0`,
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="failure"} 0`,
		`yacydhtsearch_word_partition_documents_listed_count{asked_for="matched and held documents"} 0`,
	})
}

func performedReplicaAsks() replicaasks.PerformedReplicaAsks {
	return replicaasks.PerformedReplicaAsks{
		AskedFor:  peerasks.MatchedAndHeldDocuments,
		EndedBy:   replicaasks.EndedByCoverage,
		TimeSpent: 250 * time.Millisecond,
		WordPartitions: []replicaasks.SettledWordPartition{
			{
				SettledBy:               replicaasks.SettledByCoverage,
				CoveringAskPutOn:        replicaasks.PutOnStart,
				AmountOfDocumentsListed: 5,
				AsksPutOn:               []replicaasks.PutOn{replicaasks.PutOnStart},
			},
			{
				SettledBy:               replicaasks.SettledByCoverage,
				CoveringAskPutOn:        replicaasks.PutOnHedgeDelay,
				AmountOfDocumentsListed: 0,
				AsksPutOn: []replicaasks.PutOn{
					replicaasks.PutOnStart,
					replicaasks.PutOnHedgeDelay,
				},
			},
			{
				SettledBy:               replicaasks.SettledByNoReplicaLeft,
				AmountOfDocumentsListed: 2,
				AsksPutOn: []replicaasks.PutOn{
					replicaasks.PutOnStart,
					replicaasks.PutOnFailure,
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
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="start",settled_by="coverage"} 1`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="hedge delay",settled_by="coverage"} 1`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="empty answer",settled_by="coverage"} 0`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="failure",settled_by="coverage"} 0`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="",settled_by="no replica left"} 1`,
		`yacydhtsearch_word_partitions_total{asked_for="matched and held documents",covering_ask_put_on="",settled_by="deadline"} 0`,
	})
}

func TestEveryReplicaAskIsCountedUnderWhatPutIt(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := replicaasksobserversprometheus.New(registry, queryBudget)

	metrics.ReplicaAsksPerformed(t.Context(), performedReplicaAsks())

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="start"} 3`,
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="hedge delay"} 1`,
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="failure"} 1`,
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="empty answer"} 0`,
	})
}

func TestTheDocumentsOfEverySettledWordPartitionAreMeasured(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := replicaasksobserversprometheus.New(registry, queryBudget)

	metrics.ReplicaAsksPerformed(t.Context(), performedReplicaAsks())

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_word_partition_documents_listed_count{asked_for="matched and held documents"} 3`,
		`yacydhtsearch_word_partition_documents_listed_sum{asked_for="matched and held documents"} 7`,
	})
}

func TestTheTimeTheReplicaAsksSpentIsMeasuredUnderWhatEndedThem(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := replicaasksobserversprometheus.New(registry, queryBudget)

	metrics.ReplicaAsksPerformed(t.Context(), performedReplicaAsks())

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_replica_asks_duration_seconds_count{asked_for="matched and held documents",ended_by="coverage"} 1`,
		`yacydhtsearch_replica_asks_duration_seconds_sum{asked_for="matched and held documents",ended_by="coverage"} 0.25`,
		`yacydhtsearch_replica_asks_duration_seconds_count{asked_for="matched and held documents",ended_by="deadline"} 0`,
	})
}

func TestEveryAskedForStartsAtZero(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	replicaasksobserversprometheus.New(registry, queryBudget)

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_replica_asks_total{asked_for="matched documents",put_on="start"} 0`,
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="start"} 0`,
		`yacydhtsearch_replica_asks_total{asked_for="cross-checked documents",put_on="start"} 0`,
		`yacydhtsearch_word_partition_documents_listed_count{asked_for="cross-checked documents"} 0`,
	})
}

func TestTheReplicaAsksAreCountedUnderWhatTheyAskedFor(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := replicaasksobserversprometheus.New(registry, queryBudget)

	metrics.ReplicaAsksPerformed(t.Context(), performedReplicaAsks())

	requirePublished(t, publishedBy(t, registry), []string{
		`yacydhtsearch_replica_asks_total{asked_for="matched and held documents",put_on="start"} 3`,
		`yacydhtsearch_replica_asks_total{asked_for="matched documents",put_on="start"} 0`,
		`yacydhtsearch_replica_asks_duration_seconds_count{asked_for="matched documents",ended_by="coverage"} 0`,
	})
}
