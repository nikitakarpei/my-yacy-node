package wordpartitionschedule

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type scheduleSlice uint

const scheduleSliceCount scheduleSlice = 1 << 10

const scheduleSliceWidth = yacymodel.MaxDHTRingPosition/
	yacymodel.DHTRingPosition(scheduleSliceCount) + 1

func scheduleSliceOf(position yacymodel.DHTRingPosition) scheduleSlice {
	return scheduleSlice(position / scheduleSliceWidth)
}
