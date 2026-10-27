package crawlconsoleassembly

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
)

const (
	EnvListenAddr    = "CRAWLCONSOLE_LISTEN_ADDR"
	EnvCrawlerAPIURL = "CRAWLCONSOLE_CRAWLER_API_URL"
	EnvTabBar        = "CRAWLCONSOLE_TAB_BAR"
	EnvOpsAddr       = "CRAWLCONSOLE_OPS_ADDR"
)

const (
	DefaultListenAddr = ":8093"
	DefaultOpsAddr    = ":9090"
)

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlerAPIURL, err := envconfig.RequiredHTTPURL(getenv, EnvCrawlerAPIURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	tabBar, err := webuistyle.TabBarFrom(getenv(EnvTabBar))
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("%s: %w", EnvTabBar, err)
	}

	return ServiceConfig{
		ListenAddr:    envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		CrawlerAPIURL: crawlerAPIURL,
		TabBar:        tabBar,
		OpsAddr:       envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
