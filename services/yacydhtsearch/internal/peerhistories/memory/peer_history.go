// Package memory holds the answers this instance observes for as long as the
// process runs, and no longer. It keeps no log and writes nothing down, so an
// instance that restarts starts from no presence at all. It is what runs when
// the deployment is given no NATS address to share a history over.
package memory

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerHistory struct {
	presence         *peerpresence.PeerPresence
	presenceObserver peerpresence.PeerPresenceObserver
}

func New(
	presenceLimits peerpresence.PeerPresenceLimits,
	presenceObserver peerpresence.PeerPresenceObserver,
) *PeerHistory {
	return &PeerHistory{
		presence:         peerpresence.PeerPresenceFrom(nil, presenceLimits, presenceObserver),
		presenceObserver: presenceObserver,
	}
}

func (h *PeerHistory) ObservedPeers(context.Context) []peerpresence.ObservedPeer {
	return h.presence.ObservedPeers()
}

func (h *PeerHistory) PeerAnswered(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
	answeredAt time.Time,
) {
	answeredPeer, answerCredited := h.presence.Credit(ctx, peerpresence.PeerAnswered{
		PeerAtAddress: peerpresence.PeerAtAddress{Hash: peer, Address: address},
		AnsweredAt:    answeredAt,
	})
	if !answerCredited {
		return
	}
	if answeredPeer.LatestAnsweredAt.Equal(answeredPeer.FirstAnsweredAt) {
		h.presenceObserver.PeerAnsweredForTheFirstTime(ctx, answeredPeer.Hash, answeredPeer.Address)

		return
	}
	h.presenceObserver.PeerEarnedPresence(ctx, answeredPeer.Hash, answeredPeer.Presence)
}

func (h *PeerHistory) PeerAdmitted(context.Context, yacymodel.Hash, int) {}

func (h *PeerHistory) PeerWentSilent(context.Context, yacymodel.Hash) {}

func (h *PeerHistory) PeerDropped(context.Context, yacymodel.Hash) {}

func (h *PeerHistory) PeersKnown(context.Context, int, int, int) {}
