// Package memory holds the answers this instance observes for as long as the
// process runs, and no longer. It keeps no log and writes nothing down, so an
// instance that restarts starts from no presence at all. It is what runs when
// the deployment is given no NATS address to share presence over.
package memory

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerPresence struct {
	accrual         *presenceaccrual.PresenceAccrual
	accrualObserver presenceaccrual.PresenceAccrualObserver
}

func New(
	accrualLimits presenceaccrual.PresenceAccrualLimits,
	accrualObserver presenceaccrual.PresenceAccrualObserver,
) *PeerPresence {
	return &PeerPresence{
		accrual:         presenceaccrual.PresenceAccrualFrom(nil, accrualLimits, accrualObserver),
		accrualObserver: accrualObserver,
	}
}

func (h *PeerPresence) EarnedPresenceOf(
	_ context.Context,
	peerAtAddress peeranswerhistory.PeerAtAddress,
) time.Duration {
	return h.accrual.EarnedPresenceOf(peerAtAddress)
}

func (h *PeerPresence) LatestAnswerOf(
	_ context.Context,
	peerAtAddress peeranswerhistory.PeerAtAddress,
) time.Time {
	return h.accrual.LatestAnswerOf(peerAtAddress)
}

func (h *PeerPresence) PeerAnswered(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
	answeredAt time.Time,
) {
	answeredPeer, answerCredited := h.accrual.Credit(ctx, peeranswerhistory.PeerAnswer{
		PeerAtAddress: peeranswerhistory.PeerAtAddress{Hash: peer, Address: address},
		AnsweredAt:    answeredAt,
	})
	if !answerCredited {
		return
	}
	if answeredPeer.LatestAnsweredAt.Equal(answeredPeer.FirstAnsweredAt) {
		h.accrualObserver.PeerAnsweredForTheFirstTime(ctx, answeredPeer.Hash, answeredPeer.Address)

		return
	}
	h.accrualObserver.PeerEarnedPresence(ctx, answeredPeer.Hash, answeredPeer.Presence)
}

func (h *PeerPresence) PeerAdmitted(context.Context, yacymodel.Hash, int) {}

func (h *PeerPresence) PeerWentSilent(context.Context, yacymodel.Hash) {}

func (h *PeerPresence) PeerDropped(context.Context, yacymodel.Hash) {}

func (h *PeerPresence) PeersKnown(context.Context, int, int, int) {}
