package distributioncycle

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const postingsPerPeerCap = 1000

type offer struct {
	Peer     yacymodel.Seed
	Postings []yacymodel.RWIPosting
}

type offerBatch struct {
	byPeer map[yacymodel.Hash]*offer
}

func newOfferBatch() *offerBatch {
	return &offerBatch{byPeer: make(map[yacymodel.Hash]*offer)}
}

func (b *offerBatch) Add(peer yacymodel.Seed, posting yacymodel.RWIPosting) bool {
	peerOffer := b.byPeer[peer.Hash]
	if peerOffer == nil {
		peerOffer = &offer{Peer: peer}
		b.byPeer[peer.Hash] = peerOffer
	}
	if len(peerOffer.Postings) >= postingsPerPeerCap {
		return false
	}

	peerOffer.Postings = append(peerOffer.Postings, posting)

	return true
}

func (b *offerBatch) Offers() []offer {
	offers := make([]offer, 0, len(b.byPeer))
	for _, peerOffer := range b.byPeer {
		offers = append(offers, *peerOffer)
	}

	return offers
}
