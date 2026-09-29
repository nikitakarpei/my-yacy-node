package main

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvListenAddr            = "SPAMPROXY_LISTEN_ADDR"
	EnvEgressProxyURL        = "EGRESS_PROXY_URL"
	EnvEgressProxyDialMode   = "SPAMPROXY_EGRESS_PROXY_DIAL_MODE"
	EnvResponseHeaderTimeout = "SPAMPROXY_RESPONSE_HEADER_TIMEOUT"
	EnvRelayIdleTimeout      = "SPAMPROXY_RELAY_IDLE_TIMEOUT"
	EnvModelPath             = "SPAMPROXY_MODEL_PATH"
	EnvPageByteCeiling       = "SPAMPROXY_PAGE_BYTE_CEILING"
	EnvMaxPagesReadAtOnce    = "SPAMPROXY_MAX_PAGES_READ_AT_ONCE"
	EnvOpsAddr               = "SPAMPROXY_OPS_ADDR"

	DialModeTunnel      = "tunnel"
	DialModeAbsoluteURL = "absolute-url"

	DefaultListenAddr            = ":8080"
	DefaultResponseHeaderTimeout = 10 * time.Second
	DefaultRelayIdleTimeout      = 30 * time.Second
	DefaultPageByteCeiling       = 1048576
	DefaultMaxPagesReadAtOnce    = 64
	DefaultOpsAddr               = ":9090"
)

type ServiceConfig struct {
	ListenAddr            string
	EgressProxyURL        *url.URL
	EgressProxyDialMode   string
	ModelPath             string
	PageByteCeiling       int
	MaxPagesReadAtOnce    int
	ResponseHeaderTimeout time.Duration
	RelayIdleTimeout      time.Duration
	OpsAddr               string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	egressProxyURL, urlErr := envconfig.RequiredHTTPURL(getenv, EnvEgressProxyURL)
	dialMode, dialModeErr := dialModeFrom(getenv)
	modelPath, modelPathErr := envconfig.Required(getenv, EnvModelPath)
	pageByteCeiling, ceilingErr := envconfig.PositiveInt(
		getenv,
		EnvPageByteCeiling,
		DefaultPageByteCeiling,
	)
	maxPagesReadAtOnce, maxPagesErr := envconfig.PositiveInt(
		getenv, EnvMaxPagesReadAtOnce, DefaultMaxPagesReadAtOnce,
	)
	responseHeaderTimeout, headerErr := envconfig.Duration(
		getenv, EnvResponseHeaderTimeout, DefaultResponseHeaderTimeout,
	)
	relayIdleTimeout, idleErr := envconfig.Duration(
		getenv,
		EnvRelayIdleTimeout,
		DefaultRelayIdleTimeout,
	)
	if err := errors.Join(
		urlErr, dialModeErr, modelPathErr, ceilingErr, maxPagesErr, headerErr, idleErr,
	); err != nil {
		return ServiceConfig{}, err
	}
	return ServiceConfig{
		ListenAddr:            envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		EgressProxyURL:        egressProxyURL,
		EgressProxyDialMode:   dialMode,
		ModelPath:             modelPath,
		PageByteCeiling:       pageByteCeiling,
		MaxPagesReadAtOnce:    maxPagesReadAtOnce,
		ResponseHeaderTimeout: responseHeaderTimeout,
		RelayIdleTimeout:      relayIdleTimeout,
		OpsAddr:               envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}

func dialModeFrom(getenv func(string) string) (string, error) {
	dialMode := envconfig.String(getenv, EnvEgressProxyDialMode, DialModeTunnel)
	if dialMode != DialModeTunnel && dialMode != DialModeAbsoluteURL {
		return "", fmt.Errorf(
			"%s: want %s or %s",
			EnvEgressProxyDialMode,
			DialModeTunnel,
			DialModeAbsoluteURL,
		)
	}
	return dialMode, nil
}
