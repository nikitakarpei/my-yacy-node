package yacypeerassembly

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/dashboardpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

var webUIURLs = webuitabs.URLs{
	Search:  "/",
	Crawls:  crawlConsolePathPrefix + "/",
	Metrics: dashboardpage.Path,
}
