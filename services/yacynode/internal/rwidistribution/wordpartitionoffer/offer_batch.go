package wordpartitionoffer

type Batch struct {
	Offers []WordPartitionOffer
	Waits  []WaitReason
}

func (b *Batch) addOffer(offer WordPartitionOffer) {
	b.Offers = append(b.Offers, offer)
}

func (b *Batch) addWait(waitReason WaitReason) {
	b.Waits = append(b.Waits, waitReason)
}
