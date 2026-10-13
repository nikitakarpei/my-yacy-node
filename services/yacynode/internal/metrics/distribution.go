package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelOutcome = "outcome"
	labelReason  = "reason"
	labelRequest = "request"

	requestPostings    = "postings"
	requestURLMetadata = "url_metadata"
)

var (
	peerAnswerSecondsBuckets = prometheus.ExponentialBucketsRange(0.01, 100, 17)
	postingsPerSendBuckets   = []float64{
		1, 10, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000,
	}
)

type DistributionMetrics struct {
	postingOffers          *prometheus.CounterVec
	postingsOffered        *prometheus.CounterVec
	urlMetadataDeliveries  *prometheus.CounterVec
	urlsDelivered          *prometheus.CounterVec
	urlsUnknownToUs        prometheus.Counter
	undeliveredPostings    prometheus.Gauge
	longestRefreshLateness prometheus.Gauge
	staleHoldersDropped    prometheus.Counter
	postingsHandedOff      prometheus.Counter
	batchesSkipped         *prometheus.CounterVec
	batchesCompleted       prometheus.Counter
	batchesAborted         *prometheus.CounterVec
	peersExcluded          prometheus.Counter
	wordPartitionsWaiting  *prometheus.GaugeVec
	peerRequestTime        *prometheus.HistogramVec
	peerSendTime           prometheus.Histogram
	postingsPerSend        prometheus.Histogram
	peersAtSendLimit       prometheus.Gauge
}

func NewDistributionMetrics(registry prometheus.Registerer) *DistributionMetrics {
	metrics := distributionMetrics()
	registry.MustRegister(metrics.collectors()...)

	return metrics
}

//nolint:funlen // one literal that lists every distribution metric
func distributionMetrics() *DistributionMetrics {
	return &DistributionMetrics{
		postingOffers: counterPerLabelFor(
			"yacynode_rwidistribution_posting_offers_sent_total",
			"Posting offers sent to peers, by offer outcome.",
			labelOutcome,
		),
		postingsOffered: counterPerLabelFor(
			"yacynode_rwidistribution_postings_offered_total",
			"RWI postings offered to peers, by offer outcome.",
			labelOutcome,
		),
		urlMetadataDeliveries: counterPerLabelFor(
			"yacynode_rwidistribution_url_metadata_deliveries_total",
			"URL metadata deliveries sent to peers, by delivery outcome.",
			labelOutcome,
		),
		urlsDelivered: counterPerLabelFor(
			"yacynode_rwidistribution_urls_delivered_total",
			"URLs delivered to peers as metadata, by delivery outcome.",
			labelOutcome,
		),
		urlsUnknownToUs: counterFor(
			"yacynode_rwidistribution_urls_unknown_to_us_total",
			"URLs a peer asked for whose metadata this node does not hold, so no delivery carried them.",
		),
		undeliveredPostings: gaugeFor(
			"yacynode_rwidistribution_undelivered_postings",
			"Postings not yet delivered to all their responsible peers.",
		),
		longestRefreshLateness: gaugeFor(
			"yacynode_rwidistribution_longest_refresh_lateness_seconds",
			"How long the most overdue word partition refresh has waited past its refresh time.",
		),
		staleHoldersDropped: counterFor(
			"yacynode_rwidistribution_stale_holders_dropped_total",
			"Holders dropped because they are no longer responsible.",
		),
		postingsHandedOff: counterFor(
			"yacynode_rwidistribution_postings_handed_off_total",
			"Postings deleted after peers closer to their DHT position accepted them.",
		),
		batchesSkipped: counterPerLabelFor(
			"yacynode_rwidistribution_batches_skipped_total",
			"Distribution batches the node did not start, by reason.",
			labelReason,
		),
		batchesCompleted: counterFor(
			"yacynode_rwidistribution_batches_completed_total",
			"Distribution batches that recorded their outcome or found no posting to send.",
		),
		batchesAborted: counterPerLabelFor(
			"yacynode_rwidistribution_batches_aborted_total",
			"Distribution batches aborted before their outcome was recorded, by reason.",
			labelReason,
		),
		peersExcluded: counterFor(
			"yacynode_rwidistribution_peers_excluded_total",
			"Peers excluded from distribution after failed offers in a row.",
		),
		wordPartitionsWaiting: gaugePerLabelFor(
			"yacynode_rwidistribution_word_partitions_waiting",
			"Word partitions not sent over the last full read of the ring, by reason.",
			labelReason,
		),
		peerRequestTime: histogramPerLabelFor(
			"yacynode_rwidistribution_peer_request_seconds",
			"How long a peer took to answer one request, by request kind.",
			peerAnswerSecondsBuckets,
			labelRequest,
		),
		peerSendTime: histogramFor(
			"yacynode_rwidistribution_peer_send_seconds",
			"How long a peer took for all postings of one batch and their URL metadata.",
			peerAnswerSecondsBuckets,
		),
		postingsPerSend: histogramFor(
			"yacynode_rwidistribution_postings_per_send",
			"Postings in one send: all postings of one batch to one peer.",
			postingsPerSendBuckets,
		),
		peersAtSendLimit: gaugeFor(
			"yacynode_rwidistribution_peers_at_send_limit",
			"Peers with as many sends in flight as they may have; word partitions that need them wait.",
		),
	}
}

func (d *DistributionMetrics) collectors() []prometheus.Collector {
	return []prometheus.Collector{
		d.postingOffers,
		d.postingsOffered,
		d.urlMetadataDeliveries,
		d.urlsDelivered,
		d.urlsUnknownToUs,
		d.undeliveredPostings,
		d.longestRefreshLateness,
		d.staleHoldersDropped,
		d.postingsHandedOff,
		d.batchesSkipped,
		d.batchesCompleted,
		d.batchesAborted,
		d.peersExcluded,
		d.wordPartitionsWaiting,
		d.peerRequestTime,
		d.peerSendTime,
		d.postingsPerSend,
		d.peersAtSendLimit,
	}
}

func counterPerLabelFor(name string, help string, label string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: help}, []string{label})
}

func counterFor(name string, help string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{Name: name, Help: help})
}

func gaugeFor(name string, help string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{Name: name, Help: help})
}

func gaugePerLabelFor(name string, help string, label string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, []string{label})
}

func histogramPerLabelFor(
	name string,
	help string,
	buckets []float64,
	label string,
) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: name, Help: help, Buckets: buckets},
		[]string{label},
	)
}

func histogramFor(name string, help string, buckets []float64) prometheus.Histogram {
	return prometheus.NewHistogram(
		prometheus.HistogramOpts{Name: name, Help: help, Buckets: buckets},
	)
}

func (d *DistributionMetrics) ObservePostingOffer(outcome string, postings int) {
	d.postingOffers.WithLabelValues(outcome).Inc()
	d.postingsOffered.WithLabelValues(outcome).Add(float64(postings))
}

func (d *DistributionMetrics) ObserveURLMetadataDelivery(outcome string, urls int) {
	d.urlMetadataDeliveries.WithLabelValues(outcome).Inc()
	d.urlsDelivered.WithLabelValues(outcome).Add(float64(urls))
}

func (d *DistributionMetrics) ObserveURLsUnknownToUs(urls int) {
	d.urlsUnknownToUs.Add(float64(urls))
}

func (d *DistributionMetrics) ObserveUndeliveredPostings(amountOfPostings int) {
	d.undeliveredPostings.Set(float64(amountOfPostings))
}

func (d *DistributionMetrics) ObserveLongestRefreshLateness(lateness time.Duration) {
	d.longestRefreshLateness.Set(lateness.Seconds())
}

func (d *DistributionMetrics) ObserveStaleHoldersDropped(amountOfStaleHolders int) {
	d.staleHoldersDropped.Add(float64(amountOfStaleHolders))
}

func (d *DistributionMetrics) ObservePostingsHandedOff(amountOfPostings int) {
	d.postingsHandedOff.Add(float64(amountOfPostings))
}

func (d *DistributionMetrics) ObserveBatchSkipped(reason string) {
	d.batchesSkipped.WithLabelValues(reason).Inc()
}

func (d *DistributionMetrics) ObserveBatchCompleted() {
	d.batchesCompleted.Inc()
}

func (d *DistributionMetrics) ObserveBatchAborted(reason string) {
	d.batchesAborted.WithLabelValues(reason).Inc()
}

func (d *DistributionMetrics) ObservePeerExcluded() {
	d.peersExcluded.Inc()
}

func (d *DistributionMetrics) ObserveWordPartitionsWaiting(
	waitReason string,
	amountOfWordPartitions int,
) {
	d.wordPartitionsWaiting.WithLabelValues(waitReason).Set(float64(amountOfWordPartitions))
}

func (d *DistributionMetrics) ObservePostingOfferAnswerTime(answerTime time.Duration) {
	d.peerRequestTime.WithLabelValues(requestPostings).Observe(answerTime.Seconds())
}

func (d *DistributionMetrics) ObserveURLMetadataDeliveryAnswerTime(answerTime time.Duration) {
	d.peerRequestTime.WithLabelValues(requestURLMetadata).Observe(answerTime.Seconds())
}

func (d *DistributionMetrics) ObservePeerSendTime(sendTime time.Duration) {
	d.peerSendTime.Observe(sendTime.Seconds())
}

func (d *DistributionMetrics) ObservePostingsPerSend(amountOfPostings int) {
	d.postingsPerSend.Observe(float64(amountOfPostings))
}

func (d *DistributionMetrics) ObservePeersAtSendLimit(amountOfPeers int) {
	d.peersAtSendLimit.Set(float64(amountOfPeers))
}
