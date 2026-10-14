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

type Cooldowns interface {
	OfferAccepted(peer yacymodel.Hash)
	OfferDeclined(peer yacymodel.Hash, requestedPause time.Duration)
}

type peerOffer struct {
	peer     yacymodel.Seed
	postings []yacymodel.RWIPosting
}

func sendOffers(
	ctx context.Context,
	transfers PostingTransfers,
	cooldowns Cooldowns,
	offers []wordpartitionoffer.WordPartitionOffer,
	postingsPerTransfer int,
) map[yacymodel.Hash]wordpartitionoffer.PeerAnswer {
	offersPerPeer := peerOffersFrom(offers)
	peerAnswers := make([]wordpartitionoffer.PeerAnswer, len(offersPerPeer))
	var sends sync.WaitGroup
	for index, offer := range offersPerPeer {
		sends.Go(func() {
			peerAnswers[index] = offer.sendInTransfersOf(
				ctx, transfers, cooldowns, postingsPerTransfer,
			)
		})
	}
	sends.Wait()

	answers := make(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer, len(offersPerPeer))
	for index, offer := range offersPerPeer {
		answers[offer.peer.Hash] = peerAnswers[index]
	}

	return answers
}

func peerOffersFrom(offers []wordpartitionoffer.WordPartitionOffer) []peerOffer {
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

func (o peerOffer) sendInTransfersOf(
	ctx context.Context,
	transfers PostingTransfers,
	cooldowns Cooldowns,
	postingsPerTransfer int,
) wordpartitionoffer.PeerAnswer {
	answer := wordpartitionoffer.PeerAnswer{Accepted: true}
	for transfer := range slices.Chunk(o.postings, postingsPerTransfer) {
		offerAnswer := transfers.Send(ctx, o.peer, transfer)
		reportAnswerTo(cooldowns, o.peer.Hash, offerAnswer)
		answer.AcceptedPostings = append(answer.AcceptedPostings, offerAnswer.AcceptedPostings...)
		if !offerAnswer.Accepted {
			answer.Accepted = false

			return answer
		}
	}

	return answer
}

func reportAnswerTo(
	cooldowns Cooldowns,
	peer yacymodel.Hash,
	answer postingtransfer.OfferAnswer,
) {
	if answer.Accepted {
		cooldowns.OfferAccepted(peer)
	} else {
		cooldowns.OfferDeclined(peer, answer.RequestedPause)
	}
}
