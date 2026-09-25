package replicaasks

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"

type SettledWordPartition[Ask any, Answered any] struct {
	AskOutcomes peerasks.AskOutcomes[Ask, Answered]
}
