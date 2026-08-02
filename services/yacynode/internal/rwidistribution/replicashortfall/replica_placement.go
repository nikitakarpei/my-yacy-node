package replicashortfall

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
)

// replicaPlacement is where one posting's replicas stand against the DHT at the
// moment the cycle reads it.
type replicaPlacement struct {
	posting          yacymodel.RWIPosting
	identity         postingschedule.Identity
	position         yacymodel.DHTPosition
	acceptingPeers   []yacymodel.Seed
	held             replicaHolders
	replicasPeersOwe int
}

func (p replicaPlacement) atRedundancy() bool {
	return len(p.held.responsible) >= p.replicasPeersOwe
}

func staleReplicasOf(placement replicaPlacement) StaleReplicas {
	stale := StaleReplicas{Posting: placement.identity, Peers: placement.held.gone}
	if placement.atRedundancy() {
		stale.Peers = append(stale.Peers, placement.held.outsideWindow...)
	}

	return stale
}
