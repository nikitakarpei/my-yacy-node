package yacynodeassembly

import "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"

type ServiceConfig = nodeconfiguration.Settings

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	return nodeconfiguration.Load(getenv)
}
