package yacypeerassembly

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionspage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/dashboardpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

func webUIURLsFor(cfg PeerConfig) webuitabs.URLs {
	urls := webuitabs.URLs{
		Search:  "/",
		Crawls:  crawlConsolePathPrefix + "/",
		Metrics: dashboardpage.Path,
	}
	if cfg.servesCompanions() {
		urls.Companions = companionspage.Path
	}
	return urls
}
