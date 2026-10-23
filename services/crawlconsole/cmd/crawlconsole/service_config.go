package main

import (
	"net/url"
	"time"
)

const crawlerCallDeadline = 10 * time.Second

type ServiceConfig struct {
	ListenAddr    string
	CrawlerAPIURL *url.URL
	OpsAddr       string
}

func (ServiceConfig) CrawlerCallDeadline() time.Duration {
	return crawlerCallDeadline
}
