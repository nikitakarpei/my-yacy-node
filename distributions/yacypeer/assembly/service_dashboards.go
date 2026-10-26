package yacypeerassembly

import (
	yacycrawlerdoc "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/doc"
	yacydhtsearchdoc "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/doc"
	yacynodedoc "github.com/nikitakarpei/yacy-rwi-node/yacynode/doc"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/dashboardpage"
)

var serviceDashboards = []dashboardpage.Dashboard{
	{Service: yacynodeService, JSONModel: yacynodedoc.GrafanaDashboard},
	{Service: yacycrawlerService, JSONModel: yacycrawlerdoc.GrafanaDashboard},
	{Service: yacydhtsearchService, JSONModel: yacydhtsearchdoc.GrafanaDashboard},
}
