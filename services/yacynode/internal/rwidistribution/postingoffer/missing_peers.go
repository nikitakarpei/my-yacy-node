package postingoffer

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type missingPeers struct {
	outOfCooldown       []yacymodel.Seed
	longestCooldownLeft time.Duration
}

func missingPeersFrom(closestPeers []yacymodel.Seed, cooldowns Cooldowns) missingPeers {
	peers := missingPeers{outOfCooldown: make([]yacymodel.Seed, 0, len(closestPeers))}
	for _, seed := range closestPeers {
		cooldownLeft := cooldowns.CooldownLeftOf(seed.Hash)
		if cooldownLeft == 0 {
			peers.outOfCooldown = append(peers.outOfCooldown, seed)
		}
		peers.longestCooldownLeft = max(peers.longestCooldownLeft, cooldownLeft)
	}

	return peers
}
