package yacydhtsearchpageassembly

import (
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvListenAddr       = "YACYDHTSEARCHPAGE_LISTEN_ADDR"
	EnvYaCyDHTSearchURL = "YACYDHTSEARCHPAGE_YACYDHTSEARCH_URL"
	EnvOpsAddr          = "YACYDHTSEARCHPAGE_OPS_ADDR"
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

	return ServiceConfig{
		ListenAddr:       envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		YaCyDHTSearchURL: yacyDHTSearchURL,
		OpsAddr:          envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
