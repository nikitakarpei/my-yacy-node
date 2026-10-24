package yacydhtsearchpageassembly

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvListenAddr       = "YACYDHTSEARCHPAGE_LISTEN_ADDR"
	EnvYaCyDHTSearchURL = "YACYDHTSEARCHPAGE_YACYDHTSEARCH_URL"
	EnvOpsAddr          = "YACYDHTSEARCHPAGE_OPS_ADDR"
	EnvVisitcrawlURL    = "YACYDHTSEARCHPAGE_VISITCRAWL_URL"
	EnvVisitLinkSecret  = "VISITCRAWL_LINK_SECRET"
)

const (
	DefaultListenAddr = ":8097"
	DefaultOpsAddr    = ":9090"
)

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	yacyDHTSearchURL, err := envconfig.RequiredHTTPURL(getenv, EnvYaCyDHTSearchURL)
	if err != nil {
		return ServiceConfig{}, err
	}

	visitRouting, err := visitRoutingFrom(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		ListenAddr:       envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		YaCyDHTSearchURL: yacyDHTSearchURL,
		VisitRouting:     visitRouting,
		OpsAddr:          envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}

func visitRoutingFrom(getenv func(string) string) (*VisitRouting, error) {
	if strings.TrimSpace(getenv(EnvVisitcrawlURL)) == "" {
		return nil, nil
	}
	visitcrawlURL, err := envconfig.RequiredHTTPURL(getenv, EnvVisitcrawlURL)
	if err != nil {
		return nil, err
	}
	linkSecret, err := envconfig.Required(getenv, EnvVisitLinkSecret)
	if err != nil {
		return nil, err
	}
	return &VisitRouting{VisitcrawlURL: visitcrawlURL, LinkSecret: linkSecret}, nil
}
