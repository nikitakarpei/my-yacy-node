package yacypeerassembly

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/dashboardpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/webuitabs"
)

func webUILinksWith(companionsLinks webuitabs.Links) webuitabs.Links {
	return slices.Concat(
		webuitabs.Links{
			{Title: webuitabs.Search, URL: "/"},
			{Title: webuitabs.Crawls, URL: crawlConsolePathPrefix + "/"},
		},
		companionsLinks,
		webuitabs.Links{{Title: webuitabs.Metrics, URL: dashboardpage.Path}},
	)
}
