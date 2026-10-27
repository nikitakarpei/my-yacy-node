package searchpageassembly

import (
	"net/url"
	"time"
)

const corpusTextCallDeadline = 5 * time.Second

type ServiceConfig struct {
	CorpusTextURL          *url.URL
	YaCySearchURL          *url.URL
	VisitCrawl             *VisitCrawlConfig
	ListenAddr             string
	OpsAddr                string
	YaCySearchCallDeadline time.Duration
}

type VisitCrawlConfig struct {
	URL          *url.URL
	LinkSecret   string
	LinkLifetime time.Duration
}

func (ServiceConfig) CorpusTextCallDeadline() time.Duration {
	return corpusTextCallDeadline
}
