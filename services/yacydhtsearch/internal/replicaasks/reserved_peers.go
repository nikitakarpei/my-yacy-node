package replicaasks

import (
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type reservedPeers struct {
	mutex sync.Mutex
	peers map[yacymodel.Hash]struct{}
}

func noReservedPeers() *reservedPeers {
	return &reservedPeers{peers: map[yacymodel.Hash]struct{}{}}
}

func (reserved *reservedPeers) reserve(peer yacymodel.Hash) bool {
	reserved.mutex.Lock()
	defer reserved.mutex.Unlock()

	if _, alreadyReserved := reserved.peers[peer]; alreadyReserved {
		return false
	}
	reserved.peers[peer] = struct{}{}

	return true
}
