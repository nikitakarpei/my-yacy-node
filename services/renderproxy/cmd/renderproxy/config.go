package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvListenAddr        = "RENDERPROXY_LISTEN_ADDR"
	EnvCDPURL            = "RENDERPROXY_CDP_URL"
	EnvEgressProxyURL    = "RENDERPROXY_EGRESS_PROXY_URL"
	EnvRenderConcurrency = "RENDERPROXY_RENDER_CONCURRENCY"
	EnvRequestDeadline   = "RENDERPROXY_REQUEST_DEADLINE"
	EnvMaxResponseBytes  = "RENDERPROXY_MAX_RESPONSE_BYTES"
	EnvOpsAddr           = "RENDERPROXY_OPS_ADDR"

	DefaultListenAddr        = ":8080"
	DefaultRenderConcurrency = 4
	DefaultRequestDeadline   = 30 * time.Second
	DefaultMaxResponseBytes  = 10 * 1024 * 1024
	DefaultOpsAddr           = ":9090"
)

type ServiceConfig struct {
	ListenAddr        string
	CDPURL            string
	EgressProxyURL    *url.URL
	RenderConcurrency int
	RequestDeadline   time.Duration
	MaxResponseBytes  int64
	OpsAddr           string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	cdpURL := strings.TrimSpace(getenv(EnvCDPURL))
	if cdpURL == "" {
		return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvCDPURL)
	}
	egressProxyURL, err := requiredProxyURL(getenv, EnvEgressProxyURL)
	if err != nil {
		return ServiceConfig{}, err
	}

	renderConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvRenderConcurrency,
		DefaultRenderConcurrency,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	requestDeadline, err := envconfig.Duration(getenv, EnvRequestDeadline, DefaultRequestDeadline)
	if err != nil {
		return ServiceConfig{}, err
	}
	maxResponseBytes, err := envconfig.PositiveInt64(
		getenv,
		EnvMaxResponseBytes,
		DefaultMaxResponseBytes,
	)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		ListenAddr:        envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		CDPURL:            cdpURL,
		EgressProxyURL:    egressProxyURL,
		RenderConcurrency: renderConcurrency,
		RequestDeadline:   requestDeadline,
		MaxResponseBytes:  maxResponseBytes,
		OpsAddr:           envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}

func requiredProxyURL(getenv func(string) string, key string) (*url.URL, error) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return nil, fmt.Errorf("%s: must be set", key)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%s: scheme must be http or https", key)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("%s: must include a host", key)
	}
	return parsed, nil
}
