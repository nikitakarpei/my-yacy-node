// Package searchmetrics owns the Prometheus metrics of the documentsearch
// domain: the outcome of every answered search, what ended its index read, the
// ring distance of the requested terms, and the unsupported options peers ask
// for.
package searchmetrics

import "github.com/prometheus/client_golang/prometheus"

const (
	labelSearchOutcome = "outcome"
	labelIndexReadStop = "stop"
	labelTermPresence  = "presence"
	labelSearchOption  = "option"

	termInIndex    = "in_index"
	termNotInIndex = "not_in_index"
)

// SearchOutcome is how one search request ended, from the requesting peer's
// point of view.
type SearchOutcome string

const (
	SearchServedWithResults SearchOutcome = "served_with_results"
	SearchServedNoResults   SearchOutcome = "served_no_results"
	SearchNetworkMismatch   SearchOutcome = "network_mismatch"
	SearchInvalidCriteria   SearchOutcome = "invalid_criteria"
	SearchDeadlineExceeded  SearchOutcome = "deadline_exceeded"
	SearchIndexFailure      SearchOutcome = "index_failure"
	SearchMetadataFailure   SearchOutcome = "metadata_failure"
)

type IndexReadStopReason string

const (
	IndexReadStoppedAtEndOfTerm      IndexReadStopReason = "end_of_term"
	IndexReadStoppedAtRelevanceBound IndexReadStopReason = "relevance_bound"
	IndexReadStoppedAtDeadline       IndexReadStopReason = "deadline"
)

type SearchMetrics struct {
	searchesPerOutcome           *prometheus.CounterVec
	indexReadsPerStopReason      *prometheus.CounterVec
	termRingFractionPerPresence  *prometheus.HistogramVec
	requestsPerUnsupportedOption *prometheus.CounterVec
}

func NewSearchMetrics(registry prometheus.Registerer) *SearchMetrics {
	searchesPerOutcome := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "yacynode_documentsearch_searches_total",
			Help: "Search requests answered, by how each one ended.",
		},
		[]string{labelSearchOutcome},
	)
	indexReadsPerStopReason := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "yacynode_documentsearch_index_reads_total",
			Help: "Index reads of an answered search, by what ended each read.",
		},
		[]string{labelIndexReadStop},
	)
	termRingFractionPerPresence := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "yacynode_documentsearch_query_term_ring_fraction",
			Help: "Fraction of the DHT ring between this node and the nearest " +
				"posting position of a requested term, by whether the index " +
				"holds the term.",
			Buckets: prometheus.ExponentialBucketsRange(1e-6, 1, 13),
		},
		[]string{labelTermPresence},
	)
	requestsPerUnsupportedOption := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "yacynode_documentsearch_unsupported_options_requested_total",
			Help: "Search options peers requested that this node accepts but ignores.",
		},
		[]string{labelSearchOption},
	)
	registry.MustRegister(
		searchesPerOutcome,
		indexReadsPerStopReason,
		termRingFractionPerPresence,
		requestsPerUnsupportedOption,
	)

	return &SearchMetrics{
		searchesPerOutcome:           searchesPerOutcome,
		indexReadsPerStopReason:      indexReadsPerStopReason,
		termRingFractionPerPresence:  termRingFractionPerPresence,
		requestsPerUnsupportedOption: requestsPerUnsupportedOption,
	}
}

func (s *SearchMetrics) ObserveSearchOutcome(outcome SearchOutcome) {
	s.searchesPerOutcome.WithLabelValues(string(outcome)).Inc()
}

func (s *SearchMetrics) ObserveIndexReadStopReason(reason IndexReadStopReason) {
	s.indexReadsPerStopReason.WithLabelValues(string(reason)).Inc()
}

func (s *SearchMetrics) ObserveTermInIndex(ringFraction float64) {
	s.termRingFractionPerPresence.WithLabelValues(termInIndex).Observe(ringFraction)
}

func (s *SearchMetrics) ObserveTermNotInIndex(ringFraction float64) {
	s.termRingFractionPerPresence.WithLabelValues(termNotInIndex).Observe(ringFraction)
}

func (s *SearchMetrics) ObserveUnsupportedOptionRequested(option string) {
	s.requestsPerUnsupportedOption.WithLabelValues(option).Inc()
}
