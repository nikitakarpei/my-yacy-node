package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvListenAddr    = "CRAWLCONSOLE_LISTEN_ADDR"
	EnvCrawlerAPIURL = "CRAWLCONSOLE_CRAWLER_API_URL"
	EnvOpsAddr       = "CRAWLCONSOLE_OPS_ADDR"
)

const (
	DefaultListenAddr = ":8080"
	DefaultOpsAddr    = ":9090"
)

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlerAPIURL, err := envconfig.RequiredHTTPURL(getenv, EnvCrawlerAPIURL)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		ListenAddr:    envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		CrawlerAPIURL: crawlerAPIURL,
		OpsAddr:       envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
