package yacydhtsearchpageassembly

import (
	"net/url"
	"time"
)

const searchCallDeadline = 15 * time.Second

type ServiceConfig struct {
	ListenAddr       string
	YaCyDHTSearchURL *url.URL
	OpsAddr          string
}

func (ServiceConfig) SearchCallDeadline() time.Duration {
	return searchCallDeadline
}
