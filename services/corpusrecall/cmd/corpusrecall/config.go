package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvNATSURL       = "NATS_URL"
	EnvOrdersSubject = "NATS_ORDERS_SUBJECT"

	EnvListenAddr       = "CORPUSRECALL_LISTEN_ADDR"
	EnvOpsAddr          = "CORPUSRECALL_OPS_ADDR"
	EnvDeadline         = "CORPUSRECALL_DEADLINE"
	EnvPollInterval     = "CORPUSRECALL_POLL_INTERVAL"
	EnvMaxInFlight      = "CORPUSRECALL_MAX_IN_FLIGHT"
	EnvMaxResponseBytes = "CORPUSRECALL_MAX_RESPONSE_BYTES"

	DefaultOrdersSubject    = "yacy.crawl.orders"
	DefaultListenAddr       = ":8092"
	DefaultOpsAddr          = ":9092"
	DefaultDeadline         = 30 * time.Second
	DefaultPollInterval     = 500 * time.Millisecond
	DefaultMaxInFlight      = 256
	DefaultMaxResponseBytes = 4 << 20
)

type ServiceConfig struct {
	NATSURL          string
	OrdersSubject    string
	ListenAddr       string
	OpsAddr          string
	Deadline         time.Duration
	PollInterval     time.Duration
	MaxInFlight      int
	MaxResponseBytes int64
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	natsURL := strings.TrimSpace(getenv(EnvNATSURL))
	if natsURL == "" {
		return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvNATSURL)
	}

	deadline, err := envconfig.Duration(getenv, EnvDeadline, DefaultDeadline)
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
		Deadline:         deadline,
		PollInterval:     pollInterval,
		MaxInFlight:      maxInFlight,
		MaxResponseBytes: maxResponseBytes,
	}, nil
}
