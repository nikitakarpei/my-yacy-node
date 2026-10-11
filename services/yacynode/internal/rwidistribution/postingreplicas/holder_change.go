package postingreplicas

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type HolderChange struct {
	StaleHolders   []yacymodel.Hash
	AcceptingPeers []yacymodel.Hash
}

func (c HolderChange) HoldersFrom(recordedHolders []yacymodel.Hash) []yacymodel.Hash {
	holders := c.keptHoldersOf(recordedHolders)
	for _, peer := range c.AcceptingPeers {
		if !slices.Contains(holders, peer) {
			holders = append(holders, peer)
		}
	}

	return holders
}

func (c HolderChange) AmountOfStaleHoldersIn(recordedHolders []yacymodel.Hash) int {
	return len(recordedHolders) - len(c.keptHoldersOf(recordedHolders))
}

func (c HolderChange) keptHoldersOf(recordedHolders []yacymodel.Hash) []yacymodel.Hash {
	keptHolders := make([]yacymodel.Hash, 0, len(recordedHolders)+len(c.AcceptingPeers))
	for _, peer := range recordedHolders {
		if !slices.Contains(c.StaleHolders, peer) {
			keptHolders = append(keptHolders, peer)
		}
	}

	return keptHolders
}
