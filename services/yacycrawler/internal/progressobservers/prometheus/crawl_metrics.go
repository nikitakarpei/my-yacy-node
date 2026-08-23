package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordersettlement"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordertraversal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

var (
	_ pagevisit.VisitProgress          = (*CrawlMetrics)(nil)
	_ ordertraversal.TraversalProgress = (*CrawlMetrics)(nil)
	_ ordersettlement.OrderProgress    = (*CrawlMetrics)(nil)
)

const (
	labelReason = "reason"
	labelDemand = "demand"
)

type CrawlMetrics struct {
	ordersReceived          prometheus.Counter
	ordersCompleted         prometheus.Counter
	ordersRedelivered       prometheus.Counter
	pagesFetched            prometheus.Counter
	scrapeRequestsPublished prometheus.Counter
	pagesDisposed           *prometheus.CounterVec
	refusalsHonored         *prometheus.CounterVec
	budgetExhaustions       prometheus.Counter
	fetchDurationSecs       prometheus.Histogram
}

func New(registry prometheus.Registerer) *CrawlMetrics {
	metrics := &CrawlMetrics{
		ordersReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacycrawler_orders_received_total",
			Help: "Crawl orders received.",
		}),
		ordersCompleted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacycrawler_orders_completed_total",
			Help: "Crawl orders acknowledged after every page reached a terminal outcome.",
		}),
		ordersRedelivered: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacycrawler_orders_redelivered_total",
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
		budgetExhaustions: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacycrawler_budget_exhaustions_total",
			Help: "Runs that reached the page budget with frontier remaining.",
		}),
		fetchDurationSecs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "yacycrawler_fetch_duration_seconds",
			Help:    "Page fetch duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	registry.MustRegister(
		metrics.ordersReceived,
		metrics.ordersCompleted,
		metrics.ordersRedelivered,
		metrics.pagesFetched,
		metrics.scrapeRequestsPublished,
		metrics.pagesDisposed,
		metrics.refusalsHonored,
		metrics.budgetExhaustions,
		metrics.fetchDurationSecs,
	)
	return metrics
}

func (m *CrawlMetrics) OrderReceived()          { m.ordersReceived.Inc() }
func (m *CrawlMetrics) OrderCompleted()         { m.ordersCompleted.Inc() }
func (m *CrawlMetrics) OrderRedelivered()       { m.ordersRedelivered.Inc() }
func (m *CrawlMetrics) PageFetched()            { m.pagesFetched.Inc() }
func (m *CrawlMetrics) ScrapeRequestPublished() { m.scrapeRequestsPublished.Inc() }

func (m *CrawlMetrics) PageDisposed(reason disposal.Reason) {
	m.pagesDisposed.WithLabelValues(string(reason)).Inc()
}

func (m *CrawlMetrics) RefusalHonored(demand refusal.Demand) {
	m.refusalsHonored.WithLabelValues(string(demand)).Inc()
}

func (m *CrawlMetrics) BudgetExhausted() { m.budgetExhaustions.Inc() }

func (m *CrawlMetrics) FetchCompleted(elapsed time.Duration) {
	m.fetchDurationSecs.Observe(elapsed.Seconds())
}
