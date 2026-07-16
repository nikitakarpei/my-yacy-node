package main

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	envNATSURL           = "NATS_URL"
	envNATSOrdersSubject = "NATS_ORDERS_SUBJECT"
	envNATSIngestSubject = "NATS_INGEST_SUBJECT"
	envNATSIngestDurable = "NATS_INGEST_DURABLE"

	defaultOrdersSubject = "yacy.crawl.orders"
	defaultIngestDurable = "yacy-node"
)

var defaultIngestSubject = yacycrawlcontract.CrawledPageSubject(
	yacycrawlcontract.PageRepresentationKindRWI,
)

type crawlConfig struct {
	NATSURL       string
	OrdersSubject string
	IngestSubject string
	IngestDurable string
}

func (c crawlConfig) Enabled() bool {
	return c.NATSURL != ""
}

func loadCrawlConfig(getenv func(string) string) crawlConfig {
	url := strings.TrimSpace(getenv(envNATSURL))
	if url == "" {
		return crawlConfig{}
	}

	return crawlConfig{
		NATSURL:       url,
		OrdersSubject: envconfig.String(getenv, envNATSOrdersSubject, defaultOrdersSubject),
		IngestSubject: envconfig.String(getenv, envNATSIngestSubject, defaultIngestSubject),
		IngestDurable: envconfig.String(getenv, envNATSIngestDurable, defaultIngestDurable),
	}
}
