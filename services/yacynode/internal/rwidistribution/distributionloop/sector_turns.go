package distributionloop

import (
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

type sectorTurns struct {
	guard           sync.Mutex
	clock           Clock
	rescanDelay     time.Duration
	nextSector      yacymodel.DHTRingSector
	sectorsInFlight []yacymodel.DHTRingSector
	delayStarts     map[yacymodel.DHTRingSector]time.Time
	cursors         map[yacymodel.DHTRingSector]wordpartitionoffer.Cursor
}

func newSectorTurns(clock Clock, rescanDelay time.Duration) *sectorTurns {
	return &sectorTurns{
		clock:       clock,
		rescanDelay: rescanDelay,
		delayStarts: make(map[yacymodel.DHTRingSector]time.Time),
		cursors:     make(map[yacymodel.DHTRingSector]wordpartitionoffer.Cursor),
	}
}

func (t *sectorTurns) claimNext() (yacymodel.DHTRingSector, bool) {
	t.guard.Lock()
	defer t.guard.Unlock()

	now := t.clock.Now()
	for range yacymodel.DHTRingSectorCount {
		sector := t.nextSector
		t.nextSector = (sector + 1) % yacymodel.DHTRingSectorCount
		if slices.Contains(t.sectorsInFlight, sector) || t.rescanDelayLeftOf(sector, now) > 0 {
			continue
		}
		t.sectorsInFlight = append(t.sectorsInFlight, sector)

		return sector, true
	}

	return 0, false
}

func (t *sectorTurns) rescanDelayLeftOf(
	sector yacymodel.DHTRingSector,
	now time.Time,
) time.Duration {
	delayStart, delayed := t.delayStarts[sector]
	if !delayed {
		return 0
	}

	return delayStart.Add(t.rescanDelay).Sub(now)
}

func (t *sectorTurns) cursorOf(sector yacymodel.DHTRingSector) wordpartitionoffer.Cursor {
	t.guard.Lock()
	defer t.guard.Unlock()

	return t.cursors[sector]
}

func (t *sectorTurns) advance(sector yacymodel.DHTRingSector, next wordpartitionoffer.Cursor) {
	t.guard.Lock()
	defer t.guard.Unlock()

	t.cursors[sector] = next
}

func (t *sectorTurns) delayRescan(sector yacymodel.DHTRingSector) {
	t.guard.Lock()
	defer t.guard.Unlock()

	t.delayStarts[sector] = t.clock.Now()
}

func (t *sectorTurns) release(sector yacymodel.DHTRingSector) {
	t.guard.Lock()
	defer t.guard.Unlock()

	t.sectorsInFlight = slices.DeleteFunc(
		t.sectorsInFlight,
		func(heldSector yacymodel.DHTRingSector) bool { return heldSector == sector },
	)
}

func (t *sectorTurns) shortestRescanDelayLeft() time.Duration {
	t.guard.Lock()
	defer t.guard.Unlock()

	now := t.clock.Now()
	shortestRescanDelayLeft := t.rescanDelay
	for sector := range t.delayStarts {
		if rescanDelayLeft := t.rescanDelayLeftOf(sector, now); rescanDelayLeft > 0 {
			shortestRescanDelayLeft = min(shortestRescanDelayLeft, rescanDelayLeft)
		}
	}

	return shortestRescanDelayLeft
}
