package main

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/replydeadlines"
)

const (
	EnvListenAddr             = "SPAMPROXY_LISTEN_ADDR"
	EnvEgressProxyURL         = "EGRESS_PROXY_URL"
	EnvEgressProxyDialMode    = "SPAMPROXY_EGRESS_PROXY_DIAL_MODE"
	EnvResponseHeaderTimeout  = "SPAMPROXY_RESPONSE_HEADER_TIMEOUT"
	EnvAssessmentBudget       = "SPAMPROXY_ASSESSMENT_BUDGET"
	EnvRelayIdleTimeout       = "SPAMPROXY_RELAY_IDLE_TIMEOUT"
	EnvModelPath              = "SPAMPROXY_MODEL_PATH"
	EnvPageByteCeiling        = "SPAMPROXY_PAGE_BYTE_CEILING"
	EnvMaxPagesAssessedAtOnce = "SPAMPROXY_MAX_PAGES_ASSESSED_AT_ONCE"
	EnvOpsAddr                = "SPAMPROXY_OPS_ADDR"

	DialModeTunnel      = "tunnel"
	DialModeAbsoluteURL = "absolute-url"

	DefaultListenAddr             = ":8080"
	DefaultResponseHeaderTimeout  = 10 * time.Second
	DefaultAssessmentBudget       = time.Second
	DefaultRelayIdleTimeout       = 30 * time.Second
	DefaultPageByteCeiling        = 1048576
	DefaultMaxPagesAssessedAtOnce = 64
	DefaultOpsAddr                = ":9090"
)

var errAssessmentBudget = fmt.Errorf(
	"want %s below %s",
	EnvAssessmentBudget,
	EnvResponseHeaderTimeout,
)

type ServiceConfig struct {
	ListenAddr             string
	EgressProxyURL         *url.URL
	EgressProxyDialMode    string
	ModelPath              string
	PageByteCeiling        int
	MaxPagesAssessedAtOnce int
	ReplyTimeouts          replydeadlines.Timeouts
	RelayIdleTimeout       time.Duration
	OpsAddr                string
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
	maxPagesAssessedAtOnce, maxPagesErr := envconfig.PositiveInt(
		getenv, EnvMaxPagesAssessedAtOnce, DefaultMaxPagesAssessedAtOnce,
	)
	replyTimeouts, timeoutsErr := replyTimeoutsFrom(getenv)
	relayIdleTimeout, idleErr := envconfig.Duration(
		getenv,
		EnvRelayIdleTimeout,
		DefaultRelayIdleTimeout,
	)
	if err := errors.Join(
		urlErr, dialModeErr, modelPathErr, ceilingErr, maxPagesErr, timeoutsErr, idleErr,
	); err != nil {
		return ServiceConfig{}, err
	}
	return ServiceConfig{
		ListenAddr:             envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		EgressProxyURL:         egressProxyURL,
		EgressProxyDialMode:    dialMode,
		ModelPath:              modelPath,
		PageByteCeiling:        pageByteCeiling,
		MaxPagesAssessedAtOnce: maxPagesAssessedAtOnce,
		ReplyTimeouts:          replyTimeouts,
		RelayIdleTimeout:       relayIdleTimeout,
		OpsAddr:                envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
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

func replyTimeoutsFrom(getenv func(string) string) (replydeadlines.Timeouts, error) {
	responseHeaderTimeout, headerErr := envconfig.Duration(
		getenv, EnvResponseHeaderTimeout, DefaultResponseHeaderTimeout,
	)
	assessmentBudget, budgetErr := envconfig.Duration(
		getenv,
		EnvAssessmentBudget,
		DefaultAssessmentBudget,
	)
	if err := errors.Join(headerErr, budgetErr); err != nil {
		return replydeadlines.Timeouts{}, err
	}
	if assessmentBudget >= responseHeaderTimeout {
		return replydeadlines.Timeouts{}, errAssessmentBudget
	}
	return replydeadlines.Timeouts{
		ResponseHeader:   responseHeaderTimeout,
		AssessmentBudget: assessmentBudget,
	}, nil
}
