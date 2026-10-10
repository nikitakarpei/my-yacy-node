package main

import (
	"errors"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvListenAddr     = "IMPERSONATEPROXY_LISTEN_ADDR"
	EnvEgressProxyURL = "EGRESS_PROXY_URL"
	EnvFetchTimeout   = "IMPERSONATEPROXY_FETCH_TIMEOUT"
	EnvDefaultReferer = "IMPERSONATEPROXY_DEFAULT_REFERER"
	EnvOpsAddr        = "IMPERSONATEPROXY_OPS_ADDR"

	DefaultListenAddr   = ":8080"
	DefaultFetchTimeout = 30 * time.Second
	DefaultOpsAddr      = ":9090"
)

type ServiceConfig struct {
	ListenAddr     string
	EgressProxyURL *url.URL
	FetchTimeout   time.Duration
	DefaultReferer string
	OpsAddr        string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	egressProxyURL, urlErr := envconfig.RequiredHTTPURL(getenv, EnvEgressProxyURL)
	fetchTimeout, timeoutErr := envconfig.Duration(getenv, EnvFetchTimeout, DefaultFetchTimeout)
	defaultReferer, refererErr := defaultRefererFrom(getenv)
	if err := errors.Join(urlErr, timeoutErr, refererErr); err != nil {
		return ServiceConfig{}, err
	}
	return ServiceConfig{
		ListenAddr:     envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		EgressProxyURL: egressProxyURL,
		FetchTimeout:   fetchTimeout,
		DefaultReferer: defaultReferer,
		OpsAddr:        envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}

func defaultRefererFrom(getenv func(string) string) (string, error) {
	if envconfig.String(getenv, EnvDefaultReferer, "") == "" {
		return "", nil
	}
	defaultReferer, err := envconfig.RequiredHTTPURL(getenv, EnvDefaultReferer)
	if err != nil {
		return "", err
	}
	return defaultReferer.String(), nil
}
