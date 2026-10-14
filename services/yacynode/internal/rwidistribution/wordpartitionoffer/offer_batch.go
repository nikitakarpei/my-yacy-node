package wordpartitionoffer

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
)

const readBudgetPerPosting = 10

type offerBatch struct {
	offers           []WordPartitionOffer
	skips            []Skip
	next             Cursor
	amountOfPostings int
	amountOfReads    int
	postingsPerBatch int
}

func offerBatchFrom(from Cursor, postingsPerBatch int) offerBatch {
	return offerBatch{next: from, postingsPerBatch: postingsPerBatch}
}

func (b *offerBatch) isFull() bool {
	return b.amountOfPostings >= b.postingsPerBatch ||
		b.amountOfReads >= readBudgetPerPosting*b.postingsPerBatch
}

func (b *offerBatch) walkedPast(undelivered []undeliveredpostings.UndeliveredPosting) {
	b.amountOfReads += len(undelivered)
	b.next = Cursor{lastVisited: undelivered[len(undelivered)-1]}
}

func (b *offerBatch) reachedSectorEnd() {
	b.next = Cursor{}
}

func (b *offerBatch) add(offer WordPartitionOffer) {
	b.offers = append(b.offers, offer)
	for _, recipient := range offer.Recipients {
		b.amountOfPostings += len(recipient.Postings)
	}
}

func (b *offerBatch) skip(offerKind OfferKind, skipReason SkipReason) {
	b.skips = append(b.skips, Skip{Offer: offerKind, Reason: skipReason})
}

func (b *offerBatch) roomLeft() int {
	return b.postingsPerBatch - b.amountOfPostings
}

func (b *offerBatch) holds(wordPartition yacymodel.WordPartition) bool {
	for _, offer := range b.offers {
		if offer.WordPartition == wordPartition {
			return true
		}
	}

	return false
}
