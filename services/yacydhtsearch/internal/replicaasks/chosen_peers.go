package replicaasks

import (
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type chosenPeers struct {
	mutex sync.Mutex
	peers map[yacymodel.Hash]struct{}
}

func noChosenPeers() *chosenPeers {
	return &chosenPeers{peers: map[yacymodel.Hash]struct{}{}}
}

func (chosen *chosenPeers) choose(peer yacymodel.Hash) bool {
	chosen.mutex.Lock()
	defer chosen.mutex.Unlock()

	if _, alreadyChosen := chosen.peers[peer]; alreadyChosen {
		return false
	}
	chosen.peers[peer] = struct{}{}

	return true
}
