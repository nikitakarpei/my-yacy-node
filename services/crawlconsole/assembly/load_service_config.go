package crawlconsoleassembly

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
)

const (
	EnvListenAddr    = "CRAWLCONSOLE_LISTEN_ADDR"
	EnvCrawlerAPIURL = "CRAWLCONSOLE_CRAWLER_API_URL"
	EnvTabBar        = "CRAWLCONSOLE_TAB_BAR"
	EnvOpsAddr       = "CRAWLCONSOLE_OPS_ADDR"
	EnvPathPrefix    = "CRAWLCONSOLE_PATH_PREFIX"
)

const (
	DefaultListenAddr = ":8093"
	DefaultOpsAddr    = ":9090"
)

var errUnusablePathPrefix = errors.New("start the path prefix with a slash and end it without one")

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlerAPIURL, err := envconfig.RequiredHTTPURL(getenv, EnvCrawlerAPIURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	tabBar, err := webuistyle.TabBarFrom(getenv(EnvTabBar))
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("%s: %w", EnvTabBar, err)
	}
	pathPrefix := getenv(EnvPathPrefix)
	if !isPathPrefix(pathPrefix) {
		return ServiceConfig{}, fmt.Errorf("%s: %w", EnvPathPrefix, errUnusablePathPrefix)
	}

	return ServiceConfig{
		ListenAddr:    envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		CrawlerAPIURL: crawlerAPIURL,
		TabBar:        tabBar,
		PathPrefix:    pathPrefix,
		OpsAddr:       envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}

func isPathPrefix(pathPrefix string) bool {
	return pathPrefix == "" ||
		strings.HasPrefix(pathPrefix, "/") && !strings.HasSuffix(pathPrefix, "/")
}
