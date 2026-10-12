package wordpartitionoffer

type Batch struct {
	Offers []WordPartitionOffer
	Skips  []SkipReason
}

func (b *Batch) add(offer WordPartitionOffer) {
	b.Offers = append(b.Offers, offer)
}

func (b *Batch) skip(skipReason SkipReason) {
	b.Skips = append(b.Skips, skipReason)
}
