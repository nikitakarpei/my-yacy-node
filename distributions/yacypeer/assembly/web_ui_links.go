package yacypeerassembly

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/dashboardpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

func webUILinksFor(cfg PeerConfig) webuitabs.Links {
	links := webuitabs.Links{
		{Title: webuitabs.Search, URL: "/"},
		{Title: webuitabs.Crawls, URL: crawlConsolePathPrefix + "/"},
	}
	if cfg.servesCompanions() {
		links = append(links, webuitabs.Link{Title: webuitabs.Companions, URL: companionspage.Path})
	}
	return append(links, webuitabs.Link{Title: webuitabs.Metrics, URL: dashboardpage.Path})
}
