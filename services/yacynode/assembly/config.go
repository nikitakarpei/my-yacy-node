package yacynodeassembly

import "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"

const (
	EnvPageOfferNATSURL         = nodeconfiguration.EnvPageOfferNATSURL
	EnvPageOfferDurable         = nodeconfiguration.EnvPageOfferDurable
	EnvAcceptRemoteIndex        = nodeconfiguration.EnvAcceptRemoteIndex
	EnvDataDir                  = nodeconfiguration.EnvDataDir
	EnvNetworkName              = nodeconfiguration.EnvNetworkName
	EnvSeedlistURLs             = nodeconfiguration.EnvSeedlistURLs
	EnvOpsAddr                  = nodeconfiguration.EnvOpsAddr
	EnvEgressProxyURL           = nodeconfiguration.EnvEgressProxyURL
	EnvAdvertiseHost            = nodeconfiguration.EnvAdvertiseHost
	EnvAdvertiseAdditionalHosts = nodeconfiguration.EnvAdvertiseAdditionalHosts
)

type ServiceConfig = nodeconfiguration.Settings

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	return nodeconfiguration.Load(getenv)
}
