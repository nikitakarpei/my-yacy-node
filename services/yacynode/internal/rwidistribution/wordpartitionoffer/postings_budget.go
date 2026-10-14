package wordpartitionoffer

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type postingsBudget struct {
	postingsPerPeer  map[yacymodel.Hash]int
	amountOfPostings int
}

func postingsBudgetOf(amountOfPostings int) *postingsBudget {
	return &postingsBudget{
		postingsPerPeer:  make(map[yacymodel.Hash]int),
		amountOfPostings: amountOfPostings,
	}
}

func (b *postingsBudget) wouldOverflowWith(offer Offer) bool {
	for _, recipient := range offer.Recipients {
		if b.postingsPerPeer[recipient.Peer.Hash]+len(recipient.Postings) > b.amountOfPostings {
			return true
		}
	}

	return false
}

func (b *postingsBudget) take(offer Offer) {
	for _, recipient := range offer.Recipients {
		b.postingsPerPeer[recipient.Peer.Hash] += len(recipient.Postings)
	}
}
