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

type postingAnswers struct {
	acceptingPeers []yacymodel.Hash
	requestedPause time.Duration
}

type offerRound map[postingidentity.Identity]postingAnswers

func offerDuePostings(
	ctx context.Context,
	transfers PostingTransfers,
	answers OfferAnswers,
	offers []postingoffer.PostingOffer,
) offerRound {
	offersPerPeer := peerOffersFrom(offers)
	offerAnswers := sendOffers(ctx, transfers, offersPerPeer)

	round := make(offerRound, len(offers))
	for index, offer := range offersPerPeer {
		reportAnswerTo(answers, offer, offerAnswers[index])
		round.record(offer, offerAnswers[index])
	}

	return round
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
	transfers PostingTransfers,
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

func reportAnswerTo(answers OfferAnswers, offer peerOffer, answer postingtransfer.OfferAnswer) {
	if answer.Accepted {
		answers.OfferAccepted(offer.peer.Hash)
	} else {
		answers.OfferDeclined(offer.peer.Hash, answer.RequestedPause)
	}
}

func (r offerRound) record(offer peerOffer, answer postingtransfer.OfferAnswer) {
	for _, posting := range offer.postings {
		identity := postingidentity.IdentityOf(posting)
		answersOfPosting := r[identity]
		answersOfPosting.requestedPause = max(
			answersOfPosting.requestedPause,
			answer.RequestedPause,
		)
		r[identity] = answersOfPosting
	}
	for _, posting := range answer.AcceptedPostings {
		identity := postingidentity.IdentityOf(posting)
		answersOfPosting := r[identity]
		answersOfPosting.acceptingPeers = append(answersOfPosting.acceptingPeers, offer.peer.Hash)
		r[identity] = answersOfPosting
	}
}
