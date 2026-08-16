package main

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
)

const version = "1.83"

func nodeIdentity(
	config nodeconfiguration.IdentityConfig,
	now func() time.Time,
) nodeidentity.Identity {
	return nodeidentity.Identity{
		Hash:        config.Hash,
		NetworkName: config.NetworkName,
		Name:        config.Name,
		Host:        config.AdvertiseHost,
		Port:        config.AdvertisePort,
		Flags:       config.Flags,
		Version:     version,
		Start:       now(),
	}
}
