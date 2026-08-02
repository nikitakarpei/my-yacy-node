package distributioncycle

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
)

type OfferAnswers interface {
	OfferAccepted(peer yacymodel.Hash)
	OfferDeclined(peer yacymodel.Hash, requestedPause time.Duration)
}

type peerOffer struct {
	peer     yacymodel.Seed
	postings []yacymodel.RWIPosting
}

type peerAcceptance struct {
	holder   yacymodel.Hash
	postings []yacymodel.RWIPosting
}

type requestedPauses map[postingidentity.Identity]time.Duration

func (p requestedPauses) record(postings []yacymodel.RWIPosting, pause time.Duration) {
	for _, posting := range postings {
		identity := postingidentity.IdentityOf(posting.WordHash, posting.URLHash)
		if pause > p[identity] {
			p[identity] = pause
		}
	}
}

type offerRound struct {
	acceptances []peerAcceptance
	pauses      requestedPauses
}

func offerDuePostings(
	ctx context.Context,
	transfers *postingtransfer.PostingTransfers,
	answers OfferAnswers,
	offers []postingoffer.PostingOffer,
) offerRound {
	pauses := make(requestedPauses)

	var acceptances []peerAcceptance
	for _, offer := range peerOffers(offers) {
		postingsAcceptedByPeer := sendOffer(ctx, transfers, answers, offer, pauses)
		if len(postingsAcceptedByPeer) == 0 {
			continue
		}
		acceptances = append(
			acceptances,
			peerAcceptance{holder: offer.peer.Hash, postings: postingsAcceptedByPeer},
		)
	}

	return offerRound{acceptances: acceptances, pauses: pauses}
}

func peerOffers(offers []postingoffer.PostingOffer) []peerOffer {
	byPeer := make(map[yacymodel.Hash]*peerOffer)
	order := make([]yacymodel.Hash, 0, len(offers))
	for _, offer := range offers {
		for _, peer := range offer.Peers {
			toPeer, peerGrouped := byPeer[peer.Hash]
			if !peerGrouped {
				toPeer = &peerOffer{peer: peer}
				byPeer[peer.Hash] = toPeer
				order = append(order, peer.Hash)
			}
			toPeer.postings = append(toPeer.postings, offer.Posting)
		}
	}

	toPeers := make([]peerOffer, 0, len(order))
	for _, peer := range order {
		toPeers = append(toPeers, *byPeer[peer])
	}

	return toPeers
}

func sendOffer(
	ctx context.Context,
	transfers *postingtransfer.PostingTransfers,
	answers OfferAnswers,
	offer peerOffer,
	pauses requestedPauses,
) []yacymodel.RWIPosting {
	answer := transfers.Send(ctx, offer.peer, offer.postings)
	if answer.Accepted {
		answers.OfferAccepted(offer.peer.Hash)
	} else {
		answers.OfferDeclined(offer.peer.Hash, answer.RequestedPause)
	}
	pauses.record(offer.postings, answer.RequestedPause)

	return answer.AcceptedPostings
}
