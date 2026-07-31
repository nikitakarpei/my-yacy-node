package rwidistribution

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const postingsPerPeerCap = 1000

type postingOffer struct {
	Peer     yacymodel.Seed
	Postings []yacymodel.RWIPosting
}

type peerOfferBatch struct {
	byPeer map[yacymodel.Hash]*postingOffer
}

func newPeerOfferBatch() *peerOfferBatch {
	return &peerOfferBatch{byPeer: make(map[yacymodel.Hash]*postingOffer)}
}

func (b *peerOfferBatch) Add(peer yacymodel.Seed, posting yacymodel.RWIPosting) bool {
	offer := b.byPeer[peer.Hash]
	if offer != nil && len(offer.Postings) >= postingsPerPeerCap {
		return false
	}
	if offer == nil {
		offer = &postingOffer{Peer: peer}
		b.byPeer[peer.Hash] = offer
	}

	offer.Postings = append(offer.Postings, posting)

	return true
}

func (b *peerOfferBatch) Offers() []postingOffer {
	offers := make([]postingOffer, 0, len(b.byPeer))
	for _, offer := range b.byPeer {
		offers = append(offers, *offer)
	}

	return offers
}
