package distributionloop

import (
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type arcPick struct {
	arc                *arc
	spans              []yacymodel.DHTRingSpan
	position           arcPosition
	amountOfDueSignals int
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
	lastPicked  *arc
	inFlight    map[yacymodel.WordPartition]struct{}
	wake        chan struct{}
}

func newArcWalk(clock Clock, eligibility PeerEligibility) *arcWalk {
	return &arcWalk{
		clock:       clock,
		eligibility: eligibility,
		inFlight:    make(map[yacymodel.WordPartition]struct{}),
		wake:        make(chan struct{}),
	}
}

func (w *arcWalk) followRing(seedArcs []yacymodel.SeedArc) {
	w.guard.Lock()
	defer w.guard.Unlock()

	if w.followsRing(seedArcs) {
		return
	}
	arcs := make([]*arc, 0, len(seedArcs))
	for _, seedArc := range seedArcs {
		arcs = append(arcs, w.arcWithEndsOf(seedArc))
	}
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

func (w *arcWalk) pickOpenArc() (pick arcPick, found bool, alarm wakeAlarm) {
	w.guard.Lock()
	defer w.guard.Unlock()

	now := w.clock.Now()
	first := slices.Index(w.arcs, w.lastPicked) + 1
	for turn := range len(w.arcs) {
		picked := w.arcs[(first+turn)%len(w.arcs)]
		if picked.isPickableAt(now, w.eligibility) {
			picked.isInBatch = true
			w.lastPicked = picked

			return arcPickOf(picked), true, wakeAlarm{}
		}
	}

	return arcPick{}, false, w.alarmAt(now)
}

func arcPickOf(picked *arc) arcPick {
	return arcPick{
		arc:                picked,
		spans:              picked.seedArc.Spans(),
		position:           picked.position,
		amountOfDueSignals: picked.amountOfDueSignals,
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

func (w *arcWalk) reserve(entries []readEntry) []readEntry {
	w.guard.Lock()
	defer w.guard.Unlock()

	var reserved []readEntry
	for _, entry := range entries {
		if _, isInFlight := w.inFlight[entry.scheduled.WordPartition]; !isInFlight {
			w.inFlight[entry.scheduled.WordPartition] = struct{}{}
			reserved = append(reserved, entry)
		}
	}

	return reserved
}

func (w *arcWalk) release(wordPartitions []yacymodel.WordPartition) {
	w.guard.Lock()
	defer w.guard.Unlock()

	for _, wordPartition := range wordPartitions {
		delete(w.inFlight, wordPartition)
	}
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
	case picked.amountOfDueSignals != pick.amountOfDueSignals:
	case !isAnyDue:
		picked.drainUntilSignal()
	case earliestDueAt.After(w.clock.Now()):
		picked.drainUntil(earliestDueAt)
	case pick.position == arcPosition{}:
		picked.isHeldBack = true
	}
	w.wakeWorkers()
}

func (w *arcWalk) complete(picked *arc, wordPartitions []yacymodel.WordPartition) {
	w.guard.Lock()
	defer w.guard.Unlock()

	picked.isInBatch = false
	for _, wordPartition := range wordPartitions {
		delete(w.inFlight, wordPartition)
	}
	for _, held := range w.arcs {
		held.isHeldBack = false
	}
	w.wakeWorkers()
}

func (w *arcWalk) abandon(picked *arc, wordPartitions []yacymodel.WordPartition) {
	w.guard.Lock()
	defer w.guard.Unlock()

	picked.isInBatch = false
	picked.position = arcPosition{}
	for _, wordPartition := range wordPartitions {
		delete(w.inFlight, wordPartition)
	}
}

func (w *arcWalk) signalDue(position yacymodel.DHTRingPosition) {
	w.guard.Lock()
	defer w.guard.Unlock()

	for _, holding := range w.arcs {
		if holding.holds(position) {
			holding.receiveDueSignal()
		}
	}
	w.wakeWorkers()
}

func (w *arcWalk) signalReachablePeers() {
	w.guard.Lock()
	defer w.guard.Unlock()

	w.wakeWorkers()
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
