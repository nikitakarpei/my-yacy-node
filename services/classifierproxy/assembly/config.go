package classifierproxyassembly

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvListenAddr            = "CLASSIFIERPROXY_LISTEN_ADDR"
	EnvEgressProxyURL        = "EGRESS_PROXY_URL"
	EnvEgressProxyDialMode   = "CLASSIFIERPROXY_EGRESS_PROXY_DIAL_MODE"
	EnvResponseHeaderTimeout = "CLASSIFIERPROXY_RESPONSE_HEADER_TIMEOUT"
	EnvRelayIdleTimeout      = "CLASSIFIERPROXY_RELAY_IDLE_TIMEOUT"
	EnvSafetensorsModelPaths = "CLASSIFIERPROXY_SAFETENSORS_MODEL_PATHS"
	EnvPageByteCeiling       = "CLASSIFIERPROXY_PAGE_BYTE_CEILING"
	EnvMaxPagesReadAtOnce    = "CLASSIFIERPROXY_MAX_PAGES_READ_AT_ONCE"
	EnvOpsAddr               = "CLASSIFIERPROXY_OPS_ADDR"

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
	ListenAddr                      string
	EgressProxyURL                  *url.URL
	EgressProxyDialMode             string
	SafetensorsModelPathPerCategory map[pageclassification.Category]string
	PageByteCeiling                 int
	MaxPagesReadAtOnce              int
	ResponseHeaderTimeout           time.Duration
	RelayIdleTimeout                time.Duration
	OpsAddr                         string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	egressProxyURL, urlErr := envconfig.RequiredHTTPURL(getenv, EnvEgressProxyURL)
	dialMode, dialModeErr := dialModeFrom(getenv)
	safetensorsModelPathPerCategory, safetensorsModelPathsErr := safetensorsModelPathsFrom(getenv)
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
		urlErr, dialModeErr, safetensorsModelPathsErr, ceilingErr, maxPagesErr, headerErr, idleErr,
	); err != nil {
		return ServiceConfig{}, err
	}
	return ServiceConfig{
		ListenAddr:                      envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		EgressProxyURL:                  egressProxyURL,
		EgressProxyDialMode:             dialMode,
		SafetensorsModelPathPerCategory: safetensorsModelPathPerCategory,
		PageByteCeiling:                 pageByteCeiling,
		MaxPagesReadAtOnce:              maxPagesReadAtOnce,
		ResponseHeaderTimeout:           responseHeaderTimeout,
		RelayIdleTimeout:                relayIdleTimeout,
		OpsAddr:                         envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}

func safetensorsModelPathsFrom(
	getenv func(string) string,
) (map[pageclassification.Category]string, error) {
	entries := envconfig.List(getenv, EnvSafetensorsModelPaths)
	if len(entries) == 0 {
		return nil, fmt.Errorf("%s: must be set", EnvSafetensorsModelPaths)
	}
	safetensorsModelPathPerCategory := make(map[pageclassification.Category]string, len(entries))
	for _, entry := range entries {
		category, safetensorsModelPath, err := categoryAndSafetensorsModelPathFrom(entry)
		if err != nil {
			return nil, err
		}
		if _, named := safetensorsModelPathPerCategory[category]; named {
			return nil, fmt.Errorf(
				"%s: category %q named twice",
				EnvSafetensorsModelPaths,
				category,
			)
		}
		safetensorsModelPathPerCategory[category] = safetensorsModelPath
	}
	return safetensorsModelPathPerCategory, nil
}

func categoryAndSafetensorsModelPathFrom(
	entry string,
) (pageclassification.Category, string, error) {
	name, safetensorsModelPath, _ := strings.Cut(entry, "=")
	category, err := pageclassification.CategoryFrom(strings.TrimSpace(name))
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", EnvSafetensorsModelPaths, err)
	}
	if safetensorsModelPath = strings.TrimSpace(safetensorsModelPath); safetensorsModelPath == "" {
		return "", "", fmt.Errorf("%s: want %s=<path>", EnvSafetensorsModelPaths, category)
	}
	return category, safetensorsModelPath, nil
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
