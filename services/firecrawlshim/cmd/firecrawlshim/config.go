package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvListenAddr    = "FIRECRAWLSHIM_LISTEN_ADDR"
	EnvRecallTarget  = "FIRECRAWLSHIM_RECALL_TARGET"
	EnvRecallTimeout = "FIRECRAWLSHIM_RECALL_TIMEOUT"

	DefaultListenAddr    = ":8093"
	DefaultRecallTimeout = 30 * time.Second
)

type ServiceConfig struct {
	ListenAddr    string
	RecallTarget  string
	RecallTimeout time.Duration
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	recallTarget := strings.TrimSpace(getenv(EnvRecallTarget))
	if recallTarget == "" {
		return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvRecallTarget)
	}

	recallTimeout, err := envconfig.Duration(getenv, EnvRecallTimeout, DefaultRecallTimeout)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		ListenAddr:    envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		RecallTarget:  recallTarget,
		RecallTimeout: recallTimeout,
	}, nil
}
