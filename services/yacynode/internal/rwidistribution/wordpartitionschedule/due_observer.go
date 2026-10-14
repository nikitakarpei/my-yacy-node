package wordpartitionschedule

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type DueObserver interface {
	ObserveWordPartitionDue(position yacymodel.DHTRingPosition)
}

type discardDueObserver struct{}

func (discardDueObserver) ObserveWordPartitionDue(yacymodel.DHTRingPosition) {}

var DiscardDueObserver DueObserver = discardDueObserver{}
