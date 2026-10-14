package distributionloop

import (
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type arcPick struct {
	arc                      *arc
	spans                    []yacymodel.DHTRingSpan
	position                 arcPosition
	amountOfAdmissionSignals int
}

type wakeAlarm struct {
	at    time.Time
	isSet bool
}

type arcWalk struct {
	guard       sync.Mutex
	clock       Clock
	eligibility PeerEligibility
	arcs        []*arc
	leftArcs    []*arc
	lastPicked  *arc
	wake        chan struct{}
}

func newArcWalk(clock Clock, eligibility PeerEligibility) *arcWalk {
	return &arcWalk{
		clock:       clock,
		eligibility: eligibility,
		wake:        make(chan struct{}),
	}
}

func (w *arcWalk) followRing(seedArcs []yacymodel.SeedArc) {
	w.guard.Lock()
	defer w.guard.Unlock()

	w.leftArcs = slices.DeleteFunc(w.leftArcs, func(left *arc) bool { return !left.isInBatch })
	if w.followsRing(seedArcs) {
		return
	}
	arcs := make([]*arc, 0, len(seedArcs))
	for _, seedArc := range seedArcs {
		arcs = append(arcs, w.arcWithEndsOf(seedArc))
	}
	w.leftArcs = append(w.leftArcs, arcsInBatchLeftOutOf(w.arcs, arcs)...)
	w.arcs = arcs
}

func (w *arcWalk) followsRing(seedArcs []yacymodel.SeedArc) bool {
	return slices.EqualFunc(w.arcs, seedArcs, func(a *arc, seedArc yacymodel.SeedArc) bool {
		return a.hasEnds(seedArc) && a.hasWindowOf(seedArc)
	})
}

func (w *arcWalk) arcWithEndsOf(seedArc yacymodel.SeedArc) *arc {
	for _, existing := range w.arcs {
		if existing.hasEnds(seedArc) {
			existing.seedArc = seedArc

			return existing
		}
	}

	return &arc{seedArc: seedArc}
}

func arcsInBatchLeftOutOf(previousArcs, arcs []*arc) []*arc {
	var leftArcs []*arc
	for _, previous := range previousArcs {
		if previous.isInBatch && !slices.Contains(arcs, previous) {
			leftArcs = append(leftArcs, previous)
		}
	}

	return leftArcs
}

func (w *arcWalk) pickOpenArc() (pick arcPick, found bool, alarm wakeAlarm) {
	w.guard.Lock()
	defer w.guard.Unlock()

	now := w.clock.Now()
	first := slices.Index(w.arcs, w.lastPicked) + 1
	for turn := range len(w.arcs) {
		picked := w.arcs[(first+turn)%len(w.arcs)]
		if w.isOpenAt(picked, now) {
			picked.isInBatch = true
			w.lastPicked = picked

			return arcPickOf(picked), true, wakeAlarm{}
		}
	}

	return arcPick{}, false, w.alarmAt(now)
}

func (w *arcWalk) isOpenAt(candidate *arc, now time.Time) bool {
	return candidate.isPickableAt(now, w.eligibility) && !w.overlapsALeftArcInBatch(candidate)
}

func (w *arcWalk) overlapsALeftArcInBatch(candidate *arc) bool {
	return slices.ContainsFunc(w.leftArcs, func(left *arc) bool {
		return left.isInBatch && left.overlaps(candidate)
	})
}

func arcPickOf(picked *arc) arcPick {
	return arcPick{
		arc:                      picked,
		spans:                    picked.seedArc.Spans(),
		position:                 picked.position,
		amountOfAdmissionSignals: picked.amountOfAdmissionSignals,
	}
}

func (w *arcWalk) alarmAt(now time.Time) wakeAlarm {
	var alarm wakeAlarm
	for _, waiting := range w.arcs {
		if waiting.isInBatch || waiting.isHeldBack || waiting.isDrainedUntilSignal {
			continue
		}
		if drainEnd, found := waiting.drainEndAt(now); found && waiting.isOpenFor(w.eligibility) {
			alarm = alarm.earlier(drainEnd)
		}
		if pauseEnd, found := waiting.pauseEndAt(w.eligibility); found {
			alarm = alarm.earlier(pauseEnd)
		}
	}

	return alarm
}

func (a wakeAlarm) earlier(at time.Time) wakeAlarm {
	if a.isSet && !at.Before(a.at) {
		return a
	}

	return wakeAlarm{at: at, isSet: true}
}

func (w *arcWalk) moveTo(picked *arc, position arcPosition) {
	w.guard.Lock()
	defer w.guard.Unlock()

	picked.position = position
}

func (w *arcWalk) settleReadEnd(pick arcPick, earliestDueAt time.Time, isAnyDue bool) {
	w.guard.Lock()
	defer w.guard.Unlock()

	picked := pick.arc
	picked.isInBatch = false
	picked.position = arcPosition{}
	switch {
	case picked.amountOfAdmissionSignals != pick.amountOfAdmissionSignals:
	case !isAnyDue:
		picked.drainUntilSignal()
	case earliestDueAt.After(w.clock.Now()):
		picked.drainUntil(earliestDueAt)
	case pick.position == arcPosition{}:
		picked.isHeldBack = true
	}
	w.wakeWorkers()
}

func (w *arcWalk) hasFreeWindow(picked *arc) bool {
	w.guard.Lock()
	defer w.guard.Unlock()

	return picked.isOpenFor(w.eligibility)
}

func (w *arcWalk) putBack(picked *arc) {
	w.guard.Lock()
	defer w.guard.Unlock()

	picked.isInBatch = false
}

func (w *arcWalk) complete(picked *arc) {
	w.guard.Lock()
	defer w.guard.Unlock()

	picked.isInBatch = false
	for _, held := range w.arcs {
		held.isHeldBack = false
	}
	w.wakeWorkers()
}

func (w *arcWalk) abandon(picked *arc) {
	w.guard.Lock()
	defer w.guard.Unlock()

	picked.isInBatch = false
	picked.position = arcPosition{}
}

func (w *arcWalk) signalAdmission(position yacymodel.DHTRingPosition) {
	w.guard.Lock()
	defer w.guard.Unlock()

	index := slices.IndexFunc(
		w.arcs,
		func(candidate *arc) bool { return candidate.holds(position) },
	)
	if index >= 0 && w.arcs[index].receiveAdmissionSignalAt(w.clock.Now()) {
		w.wakeWorkers()
	}
}

func (w *arcWalk) signalReachablePeers() {
	w.guard.Lock()
	defer w.guard.Unlock()

	w.wakeWorkers()
}

func (w *arcWalk) amountOfArcsPerState() map[ArcState]int {
	w.guard.Lock()
	defer w.guard.Unlock()

	amounts := make(map[ArcState]int, len(arcStates))
	for _, state := range arcStates {
		amounts[state] = 0
	}
	now := w.clock.Now()
	for _, counted := range w.arcs {
		amounts[counted.stateAt(now, w.eligibility)]++
	}

	return amounts
}

func (w *arcWalk) wakeSignal() <-chan struct{} {
	w.guard.Lock()
	defer w.guard.Unlock()

	return w.wake
}

func (w *arcWalk) wakeWorkers() {
	close(w.wake)
	w.wake = make(chan struct{})
}
