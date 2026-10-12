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
	lastPassWaiting map[*ringWalk]map[wordpartitionoffer.WaitReason]int
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
		lastPassWaiting: make(map[*ringWalk]map[wordpartitionoffer.WaitReason]int),
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
		batch, reachedEnd, err := walk.list.batchAfterLastRead(ctx, w.isInFlight)
		if err != nil {
			return reservedBatch{}, false, err
		}
		var reserved reservedBatch
		found := len(batch.wordPartitions()) > 0
		if found {
			reserved = w.reserve(walk, batch)
		}
		if reachedEnd {
			lastPassWaiting, passClosed := walk.endPassAt(w.clock.Now(), w.rescanDelay)
			w.reportLastPassOf(walk, lastPassWaiting, passClosed)
		}
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

	return reservedBatch{walk: walk, pass: walk.startBatch(), batch: batch}
}

func (w *ringWalks) complete(reserved reservedBatch, waits []wordpartitionoffer.WaitReason) {
	w.guard.Lock()
	defer w.guard.Unlock()

	for _, wordPartition := range reserved.batch.wordPartitions() {
		delete(w.inFlight, wordPartition)
	}
	lastPassWaiting, passClosed := reserved.walk.completeBatchOf(reserved.pass, waits)
	w.reportLastPassOf(reserved.walk, lastPassWaiting, passClosed)
}

func (w *ringWalks) reportLastPassOf(
	walk *ringWalk,
	lastPassWaiting map[wordpartitionoffer.WaitReason]int,
	passClosed bool,
) {
	if !passClosed {
		return
	}
	w.lastPassWaiting[walk] = lastPassWaiting
	for waitReason := range lastPassWaiting {
		w.reportedReasons[waitReason] = struct{}{}
	}
	for waitReason, amountOfWordPartitions := range w.waitingOverAllWalks() {
		w.observer.ObserveWordPartitionsWaiting(string(waitReason), amountOfWordPartitions)
	}
}

func (w *ringWalks) waitingOverAllWalks() map[wordpartitionoffer.WaitReason]int {
	waiting := make(map[wordpartitionoffer.WaitReason]int, len(w.reportedReasons))
	for waitReason := range w.reportedReasons {
		waiting[waitReason] = 0
	}
	for _, lastPassWaiting := range w.lastPassWaiting {
		for waitReason, amountOfWordPartitions := range lastPassWaiting {
			waiting[waitReason] += amountOfWordPartitions
		}
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
