package rwidistribution

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const postingsPerPeerCap = 1000

type postingOffer struct {
	Peer     yacymodel.Seed
	Postings []yacymodel.RWIPosting
}

type postingOffersByPeer struct {
	byPeer map[yacymodel.Hash]*postingOffer
}

func newPostingOffersByPeer() *postingOffersByPeer {
	return &postingOffersByPeer{byPeer: make(map[yacymodel.Hash]*postingOffer)}
}

func (b *postingOffersByPeer) Full(peer yacymodel.Hash) bool {
	offer := b.byPeer[peer]

	return offer != nil && len(offer.Postings) >= postingsPerPeerCap
}

func (b *postingOffersByPeer) Add(peer yacymodel.Seed, posting yacymodel.RWIPosting) {
	offer := b.byPeer[peer.Hash]
	if offer == nil {
		offer = &postingOffer{Peer: peer}
		b.byPeer[peer.Hash] = offer
	}

	offer.Postings = append(offer.Postings, posting)
}

func (b *postingOffersByPeer) Offers() []postingOffer {
	offers := make([]postingOffer, 0, len(b.byPeer))
	for _, offer := range b.byPeer {
		offers = append(offers, *offer)
	}

	return offers
}
