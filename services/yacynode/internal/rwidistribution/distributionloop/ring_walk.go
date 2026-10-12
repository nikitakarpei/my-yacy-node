package distributionloop

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

type passProgress struct {
	batchesOut       int
	hasEnded         bool
	amountsPerReason map[wordpartitionoffer.SkipReason]int
}

type ringWalk struct {
	list        walkedList
	currentPass int
	passes      map[int]*passProgress
	nextPassAt  time.Time
}

func newRingWalk(list walkedList) *ringWalk {
	return &ringWalk{list: list, passes: make(map[int]*passProgress)}
}

func (w *ringWalk) rescanDelayLeftAt(now time.Time) time.Duration {
	return w.nextPassAt.Sub(now)
}

func (w *ringWalk) startBatch() int {
	w.progressOf(w.currentPass).batchesOut++

	return w.currentPass
}

func (w *ringWalk) progressOf(pass int) *passProgress {
	progress, started := w.passes[pass]
	if !started {
		progress = &passProgress{amountsPerReason: make(map[wordpartitionoffer.SkipReason]int)}
		w.passes[pass] = progress
	}

	return progress
}

func (w *ringWalk) endPassAt(
	now time.Time,
	rescanDelay time.Duration,
) (map[wordpartitionoffer.SkipReason]int, bool) {
	endedPass := w.currentPass
	w.progressOf(endedPass).hasEnded = true
	w.currentPass++
	w.nextPassAt = now.Add(rescanDelay)

	return w.closePassIfDone(endedPass)
}

func (w *ringWalk) completeBatchOf(
	pass int,
	skips []wordpartitionoffer.SkipReason,
) (map[wordpartitionoffer.SkipReason]int, bool) {
	progress := w.progressOf(pass)
	progress.batchesOut--
	for _, skipReason := range skips {
		progress.amountsPerReason[skipReason]++
	}

	return w.closePassIfDone(pass)
}

func (w *ringWalk) closePassIfDone(pass int) (map[wordpartitionoffer.SkipReason]int, bool) {
	progress := w.progressOf(pass)
	if !progress.hasEnded || progress.batchesOut > 0 {
		return nil, false
	}
	delete(w.passes, pass)

	return progress.amountsPerReason, true
}
