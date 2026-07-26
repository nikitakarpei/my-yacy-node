package main

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	envNATSURL           = "NATS_URL"
	envNATSIngestSubject = "NATS_INGEST_SUBJECT"
	envNATSIngestDurable = "NATS_INGEST_DURABLE"

	defaultIngestDurable = "yacy-node"
)

var defaultIngestSubject = yacycrawlcontract.CrawledPageSubject(
	yacycrawlcontract.PageRepresentationKindRWI,
)

type crawlConfig struct {
	NATSURL       string
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
		IngestSubject: envconfig.String(getenv, envNATSIngestSubject, defaultIngestSubject),
		IngestDurable: envconfig.String(getenv, envNATSIngestDurable, defaultIngestDurable),
	}
}
