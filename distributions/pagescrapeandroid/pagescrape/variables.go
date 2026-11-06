package pagescrape

import (
	"strconv"

	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/environmenttext"
)

var serviceVariables = []environmenttext.Variable{
	{Name: pagescrapeassembly.EnvScrapeNATSURL},
	{Name: pagescrapeassembly.EnvProxyURL},
	{Name: pagescrapeassembly.EnvProxyDialMode, Default: pagescrapeassembly.DefaultProxyDialMode},
	{Name: pagescrapeassembly.EnvUserAgent, Default: pagescrapeassembly.DefaultUserAgent},
	{
		Name:    pagescrapeassembly.EnvScrapeMaxBodyBytes,
		Default: strconv.Itoa(pagescrapeassembly.DefaultScrapeMaxBodyBytes),
	},
	{
		Name:    pagescrapeassembly.EnvScrapeFetchDeadline,
		Default: pagescrapeassembly.DefaultScrapeFetchDeadline.String(),
	},
	{
		Name:    pagescrapeassembly.EnvScrapeMaxRedirectHops,
		Default: strconv.Itoa(pagescrapeassembly.DefaultScrapeMaxRedirectHops),
	},
	{
		Name:    pagescrapeassembly.EnvScrapeRequestDurable,
		Default: pagescrapeassembly.DefaultScrapeRequestDurable,
	},
	{
		Name:    pagescrapeassembly.EnvScrapeIntakeConcurrency,
		Default: strconv.Itoa(pagescrapeassembly.DefaultScrapeIntakeConcurrency),
	},
	{
		Name:    pagescrapeassembly.EnvScrapeRequestsKept,
		Default: strconv.Itoa(pagescrapeassembly.DefaultScrapeRequestsKept),
	},
	{
		Name:    pagescrapeassembly.EnvScrapeDeferralWindow,
		Default: pagescrapeassembly.DefaultScrapeDeferralWindow.String(),
	},
	{
		Name:    pagescrapeassembly.EnvScrapePageOfferMaxBytes,
		Default: pagescrapeassembly.DefaultScrapePageOfferMaxBytes,
	},
	{Name: pagescrapeassembly.EnvOpsAddr, Default: pagescrapeassembly.DefaultOpsAddr},
	{Name: applog.EnvLogLevel, Default: applog.DefaultLevel.String()},
}

func Variables() string {
	return environmenttext.TemplateOf(serviceVariables)
}
