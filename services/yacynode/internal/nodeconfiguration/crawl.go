package nodeconfiguration

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	EnvCrawlNATSURL      = "CRAWL_NATS_URL"
	EnvNATSIngestSubject = "NATS_INGEST_SUBJECT"
	EnvNATSIngestDurable = "NATS_INGEST_DURABLE"

	DefaultIngestDurable = "yacy-node"
)

var DefaultIngestSubject = yacycrawlcontract.CrawledPageSubject(
	yacycrawlcontract.PageRepresentationKindRWI,
)

type CrawlConfig struct {
	NATSURL       string
	IngestSubject string
	IngestDurable string
}

func (c CrawlConfig) Enabled() bool {
	return c.NATSURL != ""
}

func loadCrawlConfig(getenv func(string) string) CrawlConfig {
	url := strings.TrimSpace(getenv(EnvCrawlNATSURL))
	if url == "" {
		return CrawlConfig{}
	}

	return CrawlConfig{
		NATSURL:       url,
		IngestSubject: envconfig.String(getenv, EnvNATSIngestSubject, DefaultIngestSubject),
		IngestDurable: envconfig.String(getenv, EnvNATSIngestDurable, DefaultIngestDurable),
	}
}
