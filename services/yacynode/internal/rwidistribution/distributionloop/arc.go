package distributionloop

import (
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/peereligibility"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
)

type arc struct {
	seedArc              yacymodel.SeedArc
	position             arcPosition
	isInBatch            bool
	isHeldBack           bool
	drainedUntil         time.Time
	isDrainedUntilSignal bool
	amountOfDueSignals   int
}

type arcPosition struct {
	spanIndex int
	cursor    wordpartitionschedule.ScheduledWordPartition
}

func (a *arc) isPickableAt(now time.Time, eligibility PeerEligibility) bool {
	return !a.isInBatch && !a.isHeldBack && !a.isDrainedAt(now) && a.isOpenFor(eligibility)
}

func (a *arc) isDrainedAt(now time.Time) bool {
	return a.isDrainedUntilSignal || now.Before(a.drainedUntil)
}

func (a *arc) isOpenFor(eligibility PeerEligibility) bool {
	_, hasPausedPeer := a.pauseEndAt(eligibility)

	return !hasPausedPeer && !a.hasBusyPeer(eligibility)
}

func (a *arc) stateAt(now time.Time, eligibility PeerEligibility) ArcState {
	switch {
	case a.isInBatch:
		return ArcInBatch
	case a.hasPeerPausedBy(peereligibility.PauseFailed, eligibility):
		return ArcPeerFailed
	case a.hasPeerPausedBy(peereligibility.PauseDeclined, eligibility):
		return ArcPeerDeclined
	case a.hasBusyPeer(eligibility):
		return ArcPeerBusy
	case a.isDrainedAt(now):
		return ArcDrained
	default:
		return ArcWaitingForWorker
	}
}

func (a *arc) hasPeerPausedBy(cause peereligibility.PauseCause, eligibility PeerEligibility) bool {
	for _, seed := range a.seedArc.Window {
		if pause, isPaused := eligibility.PauseOf(seed.Hash); isPaused && pause.Cause == cause {
			return true
		}
	}

	return false
}

func (a *arc) hasBusyPeer(eligibility PeerEligibility) bool {
	for _, seed := range a.seedArc.Window {
		if eligibility.IsBusy(seed.Hash) {
			return true
		}
	}

	return false
}

func (a *arc) overlaps(other *arc) bool {
	for _, span := range a.seedArc.Spans() {
		for _, otherSpan := range other.seedArc.Spans() {
			if span.First <= otherSpan.Last && otherSpan.First <= span.Last {
				return true
			}
		}
	}

	return false
}

func (a *arc) hasEnds(other yacymodel.SeedArc) bool {
	return a.seedArc.After == other.After && a.seedArc.Through == other.Through
}

func (a *arc) hasWindowOf(other yacymodel.SeedArc) bool {
	return slices.EqualFunc(
		a.seedArc.Window,
		other.Window,
		func(seed, otherSeed yacymodel.Seed) bool {
			return seed.Hash == otherSeed.Hash
		},
	)
}

func (a *arc) drainEndAt(now time.Time) (drainEnd time.Time, found bool) {
	if a.isDrainedUntilSignal || !now.Before(a.drainedUntil) {
		return time.Time{}, false
	}

	return a.drainedUntil, true
}

func (a *arc) pauseEndAt(eligibility PeerEligibility) (pauseEnd time.Time, found bool) {
	for _, seed := range a.seedArc.Window {
		if pause, isPaused := eligibility.PauseOf(seed.Hash); isPaused &&
			(!found || pause.EndsAt.Before(pauseEnd)) {
			pauseEnd, found = pause.EndsAt, true
		}
	}

	return pauseEnd, found
}

func (a *arc) drainUntil(drainEnd time.Time) {
	a.drainedUntil = drainEnd
	a.isDrainedUntilSignal = false
}

func (a *arc) drainUntilSignal() {
	a.drainedUntil = time.Time{}
	a.isDrainedUntilSignal = true
}

func (a *arc) receiveDueSignalAt(now time.Time) (wasReopened bool) {
	wasReopened = a.isHeldBack || a.isDrainedAt(now)
	a.amountOfDueSignals++
	a.drainedUntil = time.Time{}
	a.isDrainedUntilSignal = false
	a.isHeldBack = false

	return wasReopened
}
