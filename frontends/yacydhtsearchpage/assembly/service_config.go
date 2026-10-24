package yacydhtsearchpageassembly

import (
	"net/url"
	"time"
)

const searchCallDeadline = 15 * time.Second

type ServiceConfig struct {
	ListenAddr       string
	YaCyDHTSearchURL *url.URL
	VisitRouting     *VisitRouting
	OpsAddr          string
}

type VisitRouting struct {
	VisitcrawlURL *url.URL
	LinkSecret    string
}

func (ServiceConfig) SearchCallDeadline() time.Duration {
	return searchCallDeadline
}
