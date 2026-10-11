package distributioncycle

import (
	"context"
	"sync"
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
		identity := postingidentity.IdentityOf(posting)
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
	offersPerPeer := peerOffersFrom(offers)
	offerAnswers := sendOffers(ctx, transfers, offersPerPeer)
	pauses := make(requestedPauses)

	var acceptances []peerAcceptance
	for index, offer := range offersPerPeer {
		postingsAcceptedByPeer := recordAnswer(answers, offer, offerAnswers[index], pauses)
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

func peerOffersFrom(offers []postingoffer.PostingOffer) []peerOffer {
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

	offersPerPeer := make([]peerOffer, 0, len(order))
	for _, peer := range order {
		offersPerPeer = append(offersPerPeer, *byPeer[peer])
	}

	return offersPerPeer
}

func sendOffers(
	ctx context.Context,
	transfers *postingtransfer.PostingTransfers,
	offersPerPeer []peerOffer,
) []postingtransfer.OfferAnswer {
	offerAnswers := make([]postingtransfer.OfferAnswer, len(offersPerPeer))
	var sends sync.WaitGroup
	for index, offer := range offersPerPeer {
		sends.Go(func() {
			offerAnswers[index] = transfers.Send(ctx, offer.peer, offer.postings)
		})
	}
	sends.Wait()

	return offerAnswers
}

func recordAnswer(
	answers OfferAnswers,
	offer peerOffer,
	answer postingtransfer.OfferAnswer,
	pauses requestedPauses,
) []yacymodel.RWIPosting {
	if answer.Accepted {
		answers.OfferAccepted(offer.peer.Hash)
	} else {
		answers.OfferDeclined(offer.peer.Hash, answer.RequestedPause)
	}
	pauses.record(offer.postings, answer.RequestedPause)

	return answer.AcceptedPostings
}
