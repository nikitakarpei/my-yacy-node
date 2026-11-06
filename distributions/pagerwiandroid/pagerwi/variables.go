package pagerwi

import (
	"strconv"

	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/environmenttext"
)

var serviceVariables = []environmenttext.Variable{
	{Name: pagerwiassembly.EnvPageOfferNATSURL},
	{Name: pagerwiassembly.EnvPageOfferDurable, Default: pagerwiassembly.DefaultPageOfferDurable},
	{
		Name:    pagerwiassembly.EnvPageOfferIntakeConcurrency,
		Default: strconv.Itoa(pagerwiassembly.DefaultPageOfferIntakeConcurrency),
	},
	{Name: pagerwiassembly.EnvPageRWINATSURL},
	{Name: pagerwiassembly.EnvPageRWIMaxBytes, Default: pagerwiassembly.DefaultPageRWIMaxBytes},
	{
		Name:    pagerwiassembly.EnvPageRWIPartitionExponent,
		Default: strconv.Itoa(pagerwiassembly.DefaultPageRWIPartitionExponent),
	},
	{Name: pagerwiassembly.EnvOpsAddr, Default: pagerwiassembly.DefaultOpsAddr},
	{Name: applog.EnvLogLevel, Default: applog.DefaultLevel.String()},
}

func Variables() string {
	return environmenttext.TemplateOf(serviceVariables)
}
