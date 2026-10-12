package distributionloop

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

type sectorWaiting struct {
	pass             int
	amountsPerReason map[wordpartitionoffer.WaitReason]int
}

type ringWalk struct {
	list           walkedList
	currentPass    int
	lastReadSector yacymodel.DHTRingSector
	sectors        [yacymodel.DHTRingSectorCount]sectorWaiting
	nextPassAt     time.Time
}

func newRingWalk(list walkedList) *ringWalk {
	return &ringWalk{list: list, currentPass: 1}
}

func (w *ringWalk) rescanDelayLeftAt(now time.Time) time.Duration {
	return w.nextPassAt.Sub(now)
}

func (w *ringWalk) batchAfterLastRead(
	ctx context.Context,
	isInFlight func(yacymodel.WordPartition) bool,
) (listBatch, bool, error) {
	return w.list.batchAfterLastRead(ctx, isInFlight)
}

func (w *ringWalk) readUpToLastRead() {
	w.readUpTo(w.list.lastReadSector())
}

func (w *ringWalk) readUpTo(sector yacymodel.DHTRingSector) {
	for readSector := w.lastReadSector; readSector <= sector; readSector++ {
		if w.sectors[readSector].pass < w.currentPass {
			w.sectors[readSector] = sectorWaiting{
				pass:             w.currentPass,
				amountsPerReason: make(map[wordpartitionoffer.WaitReason]int),
			}
		}
	}
	w.lastReadSector = sector
}

func (w *ringWalk) endPassAt(now time.Time, rescanDelay time.Duration) {
	w.readUpTo(yacymodel.MaxDHTRingSector)
	w.currentPass++
	w.lastReadSector = 0
	w.nextPassAt = now.Add(rescanDelay)
}

func (w *ringWalk) addWaitsOf(pass int, waits []wordpartitionoffer.Wait) {
	for _, wait := range waits {
		waiting := w.sectors[yacymodel.DHTRingSectorOf(wait.Position)]
		if waiting.pass == pass {
			waiting.amountsPerReason[wait.Reason]++
		}
	}
}

func (w *ringWalk) addWaitingTo(waiting map[wordpartitionoffer.WaitReason]int) {
	for _, sector := range w.sectors {
		for waitReason, amountOfWordPartitions := range sector.amountsPerReason {
			waiting[waitReason] += amountOfWordPartitions
		}
	}
}
