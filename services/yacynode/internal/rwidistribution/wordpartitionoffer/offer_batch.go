package wordpartitionoffer

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type Batch struct {
	Offers []WordPartitionOffer
	Waits  []Wait
}

type Wait struct {
	Position yacymodel.DHTRingPosition
	Reason   WaitReason
}

func (b *Batch) addOffer(offer WordPartitionOffer) {
	b.Offers = append(b.Offers, offer)
}

func (b *Batch) addWait(wait Wait) {
	b.Waits = append(b.Waits, wait)
}
