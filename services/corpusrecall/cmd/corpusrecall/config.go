package main

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvNATSURL       = "NATS_URL"
	EnvOrdersSubject = "NATS_ORDERS_SUBJECT"

	EnvListenAddr       = "CORPUSRECALL_LISTEN_ADDR"
	EnvOpsAddr          = "CORPUSRECALL_OPS_ADDR"
	EnvRecallLimit      = "CORPUSRECALL_RECALL_LIMIT"
	EnvPollInterval     = "CORPUSRECALL_POLL_INTERVAL"
	EnvMaxInFlight      = "CORPUSRECALL_MAX_IN_FLIGHT"
	EnvMaxResponseBytes = "CORPUSRECALL_MAX_RESPONSE_BYTES"

	DefaultOrdersSubject    = "yacy.crawl.orders"
	DefaultListenAddr       = ":8092"
	DefaultOpsAddr          = ":9092"
	DefaultRecallLimit      = 30 * time.Second
	DefaultPollInterval     = 500 * time.Millisecond
	DefaultMaxInFlight      = 256
	DefaultMaxResponseBytes = 4 << 20
)

type ServiceConfig struct {
	NATSURL          string
	OrdersSubject    string
	ListenAddr       string
	OpsAddr          string
	RecallLimit      time.Duration
	PollInterval     time.Duration
	MaxInFlight      int
	MaxResponseBytes int64
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	natsURL, err := envconfig.Required(getenv, EnvNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}

	recallLimit, err := envconfig.Duration(getenv, EnvRecallLimit, DefaultRecallLimit)
	if err != nil {
		return ServiceConfig{}, err
	}
	pollInterval, err := envconfig.Duration(getenv, EnvPollInterval, DefaultPollInterval)
	if err != nil {
		return ServiceConfig{}, err
	}
	maxInFlight, err := envconfig.PositiveInt(getenv, EnvMaxInFlight, DefaultMaxInFlight)
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
		NATSURL:          natsURL,
		OrdersSubject:    envconfig.String(getenv, EnvOrdersSubject, DefaultOrdersSubject),
		ListenAddr:       envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:          envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		RecallLimit:      recallLimit,
		PollInterval:     pollInterval,
		MaxInFlight:      maxInFlight,
		MaxResponseBytes: maxResponseBytes,
	}, nil
}
