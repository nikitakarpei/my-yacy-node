package distributionloop

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

func offersWithin(
	postingsBudget int,
	offers []wordpartitionoffer.Offer,
) []wordpartitionoffer.Offer {
	postingsPerPeer := make(map[yacymodel.Hash]int)
	for kept, offer := range offers {
		for _, recipient := range offer.Recipients {
			postingsPerPeer[recipient.Peer.Hash] += len(recipient.Postings)
			if kept > 0 && postingsPerPeer[recipient.Peer.Hash] > postingsBudget {
				return offers[:kept]
			}
		}
	}

	return offers
}
