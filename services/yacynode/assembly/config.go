package yacynodeassembly

import "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"

const (
	EnvPeerKey                       = nodeconfiguration.EnvPeerKey
	EnvPageRWINATSURL                = nodeconfiguration.EnvPageRWINATSURL
	EnvPageRWIDurable                = nodeconfiguration.EnvPageRWIDurable
	EnvAcceptRemoteIndex             = nodeconfiguration.EnvAcceptRemoteIndex
	EnvDataDir                       = nodeconfiguration.EnvDataDir
	EnvNetworkName                   = nodeconfiguration.EnvNetworkName
	EnvSeedlistURLs                  = nodeconfiguration.EnvSeedlistURLs
	EnvOpsAddr                       = nodeconfiguration.EnvOpsAddr
	EnvEgressProxyURL                = nodeconfiguration.EnvEgressProxyURL
	EnvAdvertiseHost                 = nodeconfiguration.EnvAdvertiseHost
	EnvAdvertiseAdditionalHosts      = nodeconfiguration.EnvAdvertiseAdditionalHosts
	EnvDistributionPartitionExponent = nodeconfiguration.EnvDistributionPartitionExponent
)

type ServiceConfig = nodeconfiguration.Settings

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	return nodeconfiguration.Load(getenv)
}
