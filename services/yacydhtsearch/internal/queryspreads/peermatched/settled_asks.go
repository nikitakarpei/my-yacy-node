package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

func settledAsksFrom(
	settledAsksAsTheySettle <-chan wordpartitionasks.SettledAsk,
) []wordpartitionasks.SettledAsk {
	var settledAsks []wordpartitionasks.SettledAsk
	for settledAsk := range settledAsksAsTheySettle {
		settledAsks = append(settledAsks, settledAsk)
	}

	return settledAsks
}
