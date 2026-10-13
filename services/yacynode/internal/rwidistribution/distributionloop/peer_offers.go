package distributionloop

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

type PeerEligibility interface {
	SendStarted(peer yacymodel.Hash)
	SendEnded(peer yacymodel.Hash, sendTime time.Duration)
	OfferAccepted(peer yacymodel.Hash)
	OfferDeclined(peer yacymodel.Hash, requestedPause time.Duration)
	OfferFailed(ctx context.Context, peer yacymodel.Hash)
}

type offerSender struct {
	transfers           PostingTransfers
	eligibility         PeerEligibility
	clock               Clock
	postingsPerTransfer int
}

type peerOffer struct {
	peer     yacymodel.Seed
	postings []yacymodel.RWIPosting
}

func (s offerSender) send(
	ctx context.Context,
	offers []wordpartitionoffer.Offer,
) map[yacymodel.Hash]wordpartitionoffer.PeerAnswer {
	offersPerPeer := peerOffersFrom(offers)
	peerAnswers := make([]wordpartitionoffer.PeerAnswer, len(offersPerPeer))
	var sends sync.WaitGroup
	for index, offer := range offersPerPeer {
		sends.Go(func() {
			peerAnswers[index] = s.sendTimed(ctx, offer)
		})
	}
	sends.Wait()

	answers := make(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer, len(offersPerPeer))
	for index, offer := range offersPerPeer {
		answers[offer.peer.Hash] = peerAnswers[index]
	}

	return answers
}

func peerOffersFrom(offers []wordpartitionoffer.Offer) []peerOffer {
	byPeer := make(map[yacymodel.Hash]*peerOffer)
	order := make([]yacymodel.Hash, 0, len(offers))
	for _, offer := range offers {
		for _, recipient := range offer.Recipients {
			if len(recipient.Postings) == 0 {
				continue
			}
			toPeer, peerGrouped := byPeer[recipient.Peer.Hash]
			if !peerGrouped {
				toPeer = &peerOffer{peer: recipient.Peer}
				byPeer[recipient.Peer.Hash] = toPeer
				order = append(order, recipient.Peer.Hash)
			}
			toPeer.postings = append(toPeer.postings, recipient.Postings...)
		}
	}

	offersPerPeer := make([]peerOffer, 0, len(order))
	for _, peer := range order {
		offersPerPeer = append(offersPerPeer, *byPeer[peer])
	}

	return offersPerPeer
}

func (s offerSender) sendTimed(
	ctx context.Context,
	offer peerOffer,
) wordpartitionoffer.PeerAnswer {
	s.eligibility.SendStarted(offer.peer.Hash)
	startedAt := s.clock.Now()
	answer := s.sendInTransfers(ctx, offer)
	s.eligibility.SendEnded(offer.peer.Hash, s.clock.Now().Sub(startedAt))

	return answer
}

func (s offerSender) sendInTransfers(
	ctx context.Context,
	offer peerOffer,
) wordpartitionoffer.PeerAnswer {
	answer := wordpartitionoffer.PeerAnswer{Accepted: true}
	for transfer := range slices.Chunk(offer.postings, s.postingsPerTransfer) {
		offerAnswer := s.transfers.Send(ctx, offer.peer, transfer)
		s.reportAnswer(ctx, offer.peer.Hash, offerAnswer)
		answer.AcceptedPostings = append(answer.AcceptedPostings, offerAnswer.AcceptedPostings...)
		if offerAnswer.Result != postingtransfer.OfferAccepted {
			answer.Accepted = false

			return answer
		}
	}

	return answer
}

func (s offerSender) reportAnswer(
	ctx context.Context,
	peer yacymodel.Hash,
	answer postingtransfer.OfferAnswer,
) {
	switch answer.Result {
	case postingtransfer.OfferAccepted:
		s.eligibility.OfferAccepted(peer)
	case postingtransfer.OfferFailed:
		s.eligibility.OfferFailed(ctx, peer)
	case postingtransfer.OfferDeclined:
		s.eligibility.OfferDeclined(peer, answer.RequestedPause)
	}
}
