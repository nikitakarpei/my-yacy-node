package distributionloop

import "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/peereligibility"

type ArcState string

const (
	ArcInBatch          ArcState = "in_batch"
	ArcPeerFailed       ArcState = ArcState(peereligibility.PauseFailed)
	ArcPeerDeclined     ArcState = ArcState(peereligibility.PauseDeclined)
	ArcPeerBusy         ArcState = "peer_busy"
	ArcDrained          ArcState = "drained"
	ArcWaitingForWorker ArcState = "waiting_for_worker"
)

var arcStates = []ArcState{
	ArcInBatch,
	ArcPeerFailed,
	ArcPeerDeclined,
	ArcPeerBusy,
	ArcDrained,
	ArcWaitingForWorker,
}
