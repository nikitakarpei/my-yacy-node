package crawlscrapebridgeassembly

import (
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvCrawlNATSURL                = "CRAWL_NATS_URL"
	EnvScrapeRequestNATSURL        = "SCRAPE_REQUEST_NATS_URL"
	EnvCrawledPageDurable          = "CRAWLED_PAGE_DURABLE"
	EnvCrawledPageRelayConcurrency = "CRAWLED_PAGE_RELAY_CONCURRENCY"
	EnvOpsAddr                     = "CRAWLSCRAPEBRIDGE_OPS_ADDR"

	DefaultCrawledPageDurable          = "crawlscrapebridge"
	DefaultCrawledPageRelayConcurrency = 4
	DefaultOpsAddr                     = ":9090"
)

type ServiceConfig struct {
	CrawlNATSURL                string
	ScrapeRequestNATSURL        string
	CrawledPageDurable          string
	CrawledPageRelayConcurrency int
	OpsAddr                     string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlNATSURL, crawlErr := envconfig.Required(getenv, EnvCrawlNATSURL)
	scrapeRequestNATSURL, scrapeRequestErr := envconfig.Required(getenv, EnvScrapeRequestNATSURL)
	relayConcurrency, concurrencyErr := envconfig.PositiveInt(
		getenv,
		EnvCrawledPageRelayConcurrency,
		DefaultCrawledPageRelayConcurrency,
	)
	if err := errors.Join(crawlErr, scrapeRequestErr, concurrencyErr); err != nil {
		return ServiceConfig{}, err
	}
	return ServiceConfig{
		CrawlNATSURL:         crawlNATSURL,
		ScrapeRequestNATSURL: scrapeRequestNATSURL,
		CrawledPageDurable: envconfig.String(
			getenv,
			EnvCrawledPageDurable,
			DefaultCrawledPageDurable,
		),
		CrawledPageRelayConcurrency: relayConcurrency,
		OpsAddr:                     envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
