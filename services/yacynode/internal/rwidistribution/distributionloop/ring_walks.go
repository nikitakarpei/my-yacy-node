package distributionloop

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

type reservedBatch struct {
	walk  *ringWalk
	pass  int
	batch listBatch
}

type ringWalks struct {
	guard           sync.Mutex
	clock           Clock
	rescanDelay     time.Duration
	observer        BatchObserver
	walks           []*ringWalk
	walkInTurn      int
	inFlight        map[yacymodel.WordPartition]struct{}
	reportedReasons map[wordpartitionoffer.WaitReason]struct{}
}

func newRingWalks(
	clock Clock,
	rescanDelay time.Duration,
	observer BatchObserver,
	lists ...walkedList,
) *ringWalks {
	walks := make([]*ringWalk, 0, len(lists))
	for _, list := range lists {
		walks = append(walks, newRingWalk(list))
	}

	return &ringWalks{
		clock:           clock,
		rescanDelay:     rescanDelay,
		observer:        observer,
		walks:           walks,
		inFlight:        make(map[yacymodel.WordPartition]struct{}),
		reportedReasons: make(map[wordpartitionoffer.WaitReason]struct{}),
	}
}

func (w *ringWalks) reserveNext(ctx context.Context) (reservedBatch, bool, error) {
	w.guard.Lock()
	defer w.guard.Unlock()

	for turn := range len(w.walks) {
		walkIndex := (w.walkInTurn + turn) % len(w.walks)
		walk := w.walks[walkIndex]
		if walk.rescanDelayLeftAt(w.clock.Now()) > 0 {
			continue
		}
		batch, found, err := w.reserveFrom(ctx, walk)
		if err != nil || found {
			w.walkInTurn = (walkIndex + 1) % len(w.walks)

			return batch, found, err
		}
	}

	return reservedBatch{}, false, nil
}

func (w *ringWalks) reserveFrom(ctx context.Context, walk *ringWalk) (reservedBatch, bool, error) {
	for {
		batch, reachedEnd, err := walk.batchAfterLastRead(ctx, w.isInFlight)
		if err != nil {
			return reservedBatch{}, false, err
		}
		var reserved reservedBatch
		found := len(batch.wordPartitions()) > 0
		if found {
			reserved = w.reserve(walk, batch)
		}
		if reachedEnd {
			walk.endPassAt(w.clock.Now(), w.rescanDelay)
		} else {
			walk.readUpToLastRead()
		}
		w.reportWaiting()
		if reachedEnd || found {
			return reserved, found, nil
		}
	}
}

func (w *ringWalks) isInFlight(wordPartition yacymodel.WordPartition) bool {
	_, inFlight := w.inFlight[wordPartition]

	return inFlight
}

func (w *ringWalks) reserve(walk *ringWalk, batch listBatch) reservedBatch {
	for _, wordPartition := range batch.wordPartitions() {
		w.inFlight[wordPartition] = struct{}{}
	}

	return reservedBatch{walk: walk, pass: walk.currentPass, batch: batch}
}

func (w *ringWalks) complete(reserved reservedBatch, waits []wordpartitionoffer.Wait) {
	w.guard.Lock()
	defer w.guard.Unlock()

	for _, wordPartition := range reserved.batch.wordPartitions() {
		delete(w.inFlight, wordPartition)
	}
	reserved.walk.addWaitsOf(reserved.pass, waits)
	w.reportWaiting()
}

func (w *ringWalks) reportWaiting() {
	waiting := w.waitingOverAllWalks()
	for waitReason := range waiting {
		w.reportedReasons[waitReason] = struct{}{}
	}
	for waitReason := range w.reportedReasons {
		w.observer.ObserveWordPartitionsWaiting(string(waitReason), waiting[waitReason])
	}
}

func (w *ringWalks) waitingOverAllWalks() map[wordpartitionoffer.WaitReason]int {
	waiting := make(map[wordpartitionoffer.WaitReason]int)
	for _, walk := range w.walks {
		walk.addWaitingTo(waiting)
	}

	return waiting
}

func (w *ringWalks) shortestRescanDelayLeft() time.Duration {
	w.guard.Lock()
	defer w.guard.Unlock()

	now := w.clock.Now()
	shortestRescanDelayLeft := w.rescanDelay
	for _, walk := range w.walks {
		if rescanDelayLeft := walk.rescanDelayLeftAt(now); rescanDelayLeft > 0 {
			shortestRescanDelayLeft = min(shortestRescanDelayLeft, rescanDelayLeft)
		}
	}

	return shortestRescanDelayLeft
}
