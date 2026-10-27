package yacypeerassembly

import (
	"net"

	crawlconsoleassembly "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/assembly"
	searchpageassembly "github.com/nikitakarpei/yacy-rwi-node/searchpage/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/dashboardpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

func webUIURLsFor(ops net.Listener, getenv func(string) string) webuitabs.URLs {
	return webuitabs.URLs{
		Search:  listenURLOf(searchpageService, searchpageassembly.EnvListenAddr, getenv),
		Crawls:  listenURLOf(crawlconsoleService, crawlconsoleassembly.EnvListenAddr, getenv),
		Metrics: loopback.URLOf(ops) + dashboardpage.Path,
	}
}

func listenURLOf(service string, listenAddrKey string, getenv func(string) string) string {
	environment := serviceenvironment.Of(service, nil, localListenAddresses, getenv)
	return loopback.URLAt(environment.Lookup(listenAddrKey))
}
