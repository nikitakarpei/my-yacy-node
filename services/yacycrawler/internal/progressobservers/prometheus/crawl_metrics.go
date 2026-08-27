package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitintake"
)

var (
	_ pagevisit.VisitProgress        = (*CrawlMetrics)(nil)
	_ visitintake.SettlementProgress = (*CrawlMetrics)(nil)
	_ orderintake.OrderProgress      = (*CrawlMetrics)(nil)
)

const (
	labelReason = "reason"
	labelDemand = "demand"
)

type CrawlMetrics struct {
	ordersReceived          prometheus.Counter
	ordersAccepted          prometheus.Counter
	ordersReturned          prometheus.Counter
	pagesFetched            prometheus.Counter
	scrapeRequestsPublished prometheus.Counter
	pagesDisposed           *prometheus.CounterVec
	refusalsHonored         *prometheus.CounterVec
	fetchDurationSecs       prometheus.Histogram
}

func New(registry prometheus.Registerer) *CrawlMetrics {
	metrics := &CrawlMetrics{
		ordersReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacycrawler_orders_received_total",
			Help: "Crawl orders received.",
		}),
		ordersAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacycrawler_orders_accepted_total",
			Help: "Crawl orders acknowledged once the order and its seeds are durable.",
		}),
		ordersReturned: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacycrawler_orders_returned_total",
			Help: "Crawl orders returned for redelivery.",
		}),
		pagesFetched: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacycrawler_pages_fetched_total",
			Help: "Pages fetched.",
		}),
		scrapeRequestsPublished: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacycrawler_scrape_requests_published_total",
			Help: "Scrape requests published.",
		}),
		pagesDisposed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "yacycrawler_pages_disposed_total",
			Help: "Pages disposed, by reason.",
		}, []string{labelReason}),
		refusalsHonored: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "yacycrawler_refusals_honored_total",
			Help: "Target refusals honored, by demand.",
		}, []string{labelDemand}),
		fetchDurationSecs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "yacycrawler_fetch_duration_seconds",
			Help:    "Page fetch duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	registry.MustRegister(
		metrics.ordersReceived,
		metrics.ordersAccepted,
		metrics.ordersReturned,
		metrics.pagesFetched,
		metrics.scrapeRequestsPublished,
		metrics.pagesDisposed,
		metrics.refusalsHonored,
		metrics.fetchDurationSecs,
	)
	return metrics
}

func (m *CrawlMetrics) OrderReceived()          { m.ordersReceived.Inc() }
func (m *CrawlMetrics) OrderAccepted()          { m.ordersAccepted.Inc() }
func (m *CrawlMetrics) OrderReturned()          { m.ordersReturned.Inc() }
func (m *CrawlMetrics) PageFetched()            { m.pagesFetched.Inc() }
func (m *CrawlMetrics) ScrapeRequestPublished() { m.scrapeRequestsPublished.Inc() }

func (m *CrawlMetrics) PageDisposed(reason disposal.Reason) {
	m.pagesDisposed.WithLabelValues(string(reason)).Inc()
}

func (m *CrawlMetrics) RefusalHonored(demand refusal.Demand) {
	m.refusalsHonored.WithLabelValues(string(demand)).Inc()
}

func (m *CrawlMetrics) FetchCompleted(elapsed time.Duration) {
	m.fetchDurationSecs.Observe(elapsed.Seconds())
}
