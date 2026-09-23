package replicaasks

import (
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askedPeers struct {
	mutex  sync.Mutex
	hashes map[yacymodel.Hash]struct{}
}

func noAskedPeers() *askedPeers {
	return &askedPeers{hashes: map[yacymodel.Hash]struct{}{}}
}

func (asked *askedPeers) reserve(peer yacymodel.Hash) bool {
	asked.mutex.Lock()
	defer asked.mutex.Unlock()

	if _, alreadyAsked := asked.hashes[peer]; alreadyAsked {
		return false
	}
	asked.hashes[peer] = struct{}{}

	return true
}
