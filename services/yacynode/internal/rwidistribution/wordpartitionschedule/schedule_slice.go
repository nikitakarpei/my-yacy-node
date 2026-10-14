package wordpartitionschedule

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type scheduleSlice uint

const scheduleSliceCount scheduleSlice = 1 << 10

const (
	scheduleSliceWidth = yacymodel.MaxDHTRingPosition/
		yacymodel.DHTRingPosition(scheduleSliceCount) + 1
	slicesPerSector = scheduleSliceCount / scheduleSlice(yacymodel.DHTRingSectorCount)
)

func scheduleSliceOf(position yacymodel.DHTRingPosition) scheduleSlice {
	return scheduleSlice(position / scheduleSliceWidth)
}

func (s scheduleSlice) sector() yacymodel.DHTRingSector {
	return yacymodel.DHTRingSector(s / slicesPerSector)
}
