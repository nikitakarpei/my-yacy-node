package yacycrawlerassembly

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const (
	EnvCrawlNATSURL       = "CRAWL_NATS_URL"
	EnvCrawlStateDir      = "CRAWL_STATE_DIR"
	EnvCrawlOrdersSubject = "CRAWL_ORDERS_SUBJECT"
	EnvCrawlOrdersDurable = "CRAWL_ORDERS_DURABLE"

	EnvProxyURL         = "YACYCRAWLER_FETCH_PROXY_URL"
	EnvProxyDialMode    = "YACYCRAWLER_FETCH_PROXY_DIAL_MODE"
	EnvFetchConcurrency = "YACYCRAWLER_FETCH_CONCURRENCY"

	EnvMaxBodyBytes  = "YACYCRAWLER_MAX_BODY_BYTES"
	EnvFetchDeadline = "YACYCRAWLER_FETCH_DEADLINE"
	EnvOpsAddr       = "YACYCRAWLER_OPS_ADDR"
	EnvAPIAddr       = "YACYCRAWLER_API_ADDR"
	EnvUserAgent     = "YACYCRAWLER_FETCH_USER_AGENT"
	EnvRecrawlGrace  = "YACYCRAWLER_RECRAWL_GRACE"

	EnvCrawlStatePebbleBlockCache            = "CRAWL_STATE_PEBBLE_BLOCK_CACHE"
	EnvCrawlStatePebbleMemtableSize          = "CRAWL_STATE_PEBBLE_MEMTABLE_SIZE"
	EnvCrawlStatePebbleCompactionConcurrency = "CRAWL_STATE_PEBBLE_COMPACTION_CONCURRENCY"
	EnvCrawlStatePebbleOpenFileLimit         = "CRAWL_STATE_PEBBLE_OPEN_FILE_LIMIT"
	EnvCrawlStatePebbleWriteConcurrency      = "CRAWL_STATE_PEBBLE_WRITE_CONCURRENCY"
)

const (
	DefaultCrawlOrdersSubject = "yacy.crawl.orders"
	DefaultCrawlOrdersDurable = "yacycrawler"
	DefaultFetchConcurrency   = 4
	DefaultMaxBodyBytes       = 2 << 20
	DefaultFetchDeadline      = 30 * time.Second
	DefaultOpsAddr            = ":9090"
	DefaultUserAgent          = "yacycrawler (+https://yacy.net)"
	DefaultProxyDialMode      = http.ProxyDialTunnelName
	DefaultRecrawlGrace       = time.Hour

	DefaultCrawlStateDir                         = "./crawl-state"
	DefaultCrawlStatePebbleBlockCache            = "64MB"
	DefaultCrawlStatePebbleMemtableSize          = "16MB"
	DefaultCrawlStatePebbleCompactionConcurrency = 1
	DefaultCrawlStatePebbleOpenFileLimit         = 1000
	DefaultCrawlStatePebbleWriteConcurrency      = 16
)

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlNATSURL, err := envconfig.Required(getenv, EnvCrawlNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	crawlStateLimits, err := loadCrawlStateLimits(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	proxyURL, err := envconfig.RequiredHTTPURL(getenv, EnvProxyURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	proxyDialMode, err := proxyDialModeFromEnv(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	limits, err := loadFetchLimits(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	recrawlGrace, err := envconfig.NonNegativeDuration(getenv, EnvRecrawlGrace, DefaultRecrawlGrace)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		CrawlNATSURL:     crawlNATSURL,
		CrawlStateDir:    envconfig.String(getenv, EnvCrawlStateDir, DefaultCrawlStateDir),
		CrawlStateLimits: crawlStateLimits,
		CrawlOrdersSubject: envconfig.String(
			getenv,
			EnvCrawlOrdersSubject,
			DefaultCrawlOrdersSubject,
		),
		CrawlOrdersDurable: envconfig.String(
			getenv,
			EnvCrawlOrdersDurable,
			DefaultCrawlOrdersDurable,
		),
		ProxyURL:         proxyURL,
		ProxyDialMode:    proxyDialMode,
		FetchConcurrency: limits.fetchConcurrency,
		MaxBodyBytes:     limits.maxBodyBytes,
		FetchDeadline:    limits.fetchDeadline,
		RecrawlGrace:     recrawlGrace,
		OpsAddr:          envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		APIAddr:          envconfig.String(getenv, EnvAPIAddr, ""),
		UserAgent:        envconfig.String(getenv, EnvUserAgent, DefaultUserAgent),
	}, nil
}

type fetchLimits struct {
	fetchConcurrency int
	maxBodyBytes     int64
	fetchDeadline    time.Duration
}

func loadFetchLimits(getenv func(string) string) (fetchLimits, error) {
	fetchConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvFetchConcurrency,
		DefaultFetchConcurrency,
	)
	if err != nil {
		return fetchLimits{}, err
	}
	maxBodyBytes, err := envconfig.PositiveInt64(getenv, EnvMaxBodyBytes, DefaultMaxBodyBytes)
	if err != nil {
		return fetchLimits{}, err
	}
	fetchDeadline, err := envconfig.Duration(getenv, EnvFetchDeadline, DefaultFetchDeadline)
	if err != nil {
		return fetchLimits{}, err
	}

	return fetchLimits{
		fetchConcurrency: fetchConcurrency,
		maxBodyBytes:     maxBodyBytes,
		fetchDeadline:    fetchDeadline,
	}, nil
}

func loadCrawlStateLimits(getenv func(string) string) (pebblevault.MachineLimits, error) {
	blockCache, err := envconfig.ByteSize(
		getenv, EnvCrawlStatePebbleBlockCache, DefaultCrawlStatePebbleBlockCache,
	)
	if err != nil {
		return pebblevault.MachineLimits{}, err
	}
	memtable, err := envconfig.ByteSize(
		getenv, EnvCrawlStatePebbleMemtableSize, DefaultCrawlStatePebbleMemtableSize,
	)
	if err != nil {
		return pebblevault.MachineLimits{}, err
	}
	compactionConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvCrawlStatePebbleCompactionConcurrency,
		DefaultCrawlStatePebbleCompactionConcurrency,
	)
	if err != nil {
		return pebblevault.MachineLimits{}, err
	}
	openFileLimit, err := envconfig.PositiveInt(
		getenv, EnvCrawlStatePebbleOpenFileLimit, DefaultCrawlStatePebbleOpenFileLimit,
	)
	if err != nil {
		return pebblevault.MachineLimits{}, err
	}
	writeConcurrency, err := envconfig.PositiveInt(
		getenv, EnvCrawlStatePebbleWriteConcurrency, DefaultCrawlStatePebbleWriteConcurrency,
	)
	if err != nil {
		return pebblevault.MachineLimits{}, err
	}

	return pebblevault.MachineLimits{
		BlockCacheBytes:       blockCache,
		MemtableBytes:         memtable,
		CompactionConcurrency: compactionConcurrency,
		OpenFileLimit:         openFileLimit,
		WriteConcurrency:      writeConcurrency,
	}, nil
}

func proxyDialModeFromEnv(getenv func(string) string) (http.ProxyDialMode, error) {
	mode, err := http.ProxyDialModeNamed(
		envconfig.String(getenv, EnvProxyDialMode, DefaultProxyDialMode),
	)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", EnvProxyDialMode, err)
	}
	return mode, nil
}
