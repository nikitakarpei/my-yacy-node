package wordpartitionoffer

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
)

type WordPartitionOffer struct {
	WordPartition       yacymodel.WordPartition
	Position            yacymodel.DHTRingPosition
	ResponsiblePeers    []yacymodel.Hash
	Holders             []yacymodel.Hash
	StaleHolders        []yacymodel.Hash
	UndeliveredPostings []undeliveredpostings.UndeliveredPosting
	IsRefresh           bool
	Recipients          []Recipient
}

type Recipient struct {
	Peer     yacymodel.Seed
	Postings []yacymodel.RWIPosting
}

type PeerAnswer struct {
	Accepted         bool
	AcceptedPostings []yacymodel.RWIPosting
}

type Outcome struct {
	Holders                []yacymodel.Hash
	HoldersChanged         bool
	IsAtRedundancy         bool
	DeliveredPostings      []undeliveredpostings.UndeliveredPosting
	AmountOfDroppedHolders int
	Acceptances            []Acceptance
}

type Acceptance struct {
	Peer             yacymodel.Hash
	AmountOfPostings int
}

type acceptedURLs map[yacymodel.URLHash]struct{}

func (o WordPartitionOffer) OutcomeFrom(answers map[yacymodel.Hash]PeerAnswer) Outcome {
	acceptedURLsPerPeer := o.acceptedURLsPerPeerFrom(answers)
	holders := o.holdersFrom(answers, acceptedURLsPerPeer)
	deliveredPostings := o.deliveredPostingsFrom(acceptedURLsPerPeer)

	return Outcome{
		Holders:        holders,
		HoldersChanged: !sameHolders(holders, o.Holders),
		IsAtRedundancy: len(holders) >= len(o.ResponsiblePeers) &&
			len(deliveredPostings) == len(o.UndeliveredPostings),
		DeliveredPostings:      deliveredPostings,
		AmountOfDroppedHolders: len(o.StaleHolders),
		Acceptances:            o.acceptancesFrom(acceptedURLsPerPeer),
	}
}

func (o WordPartitionOffer) acceptedURLsPerPeerFrom(
	answers map[yacymodel.Hash]PeerAnswer,
) map[yacymodel.Hash]acceptedURLs {
	acceptedURLsPerPeer := make(map[yacymodel.Hash]acceptedURLs)
	for peer, answer := range answers {
		if o.isRecipient(peer) || slices.Contains(o.ResponsiblePeers, peer) {
			acceptedURLsPerPeer[peer] = acceptedURLsOfWord(answer, o.WordPartition.Word)
		}
	}

	return acceptedURLsPerPeer
}

func (o WordPartitionOffer) isRecipient(peer yacymodel.Hash) bool {
	return slices.ContainsFunc(o.Recipients, func(recipient Recipient) bool {
		return recipient.Peer.Hash == peer
	})
}

func acceptedURLsOfWord(answer PeerAnswer, word yacymodel.Hash) acceptedURLs {
	urls := make(acceptedURLs)
	for _, posting := range answer.AcceptedPostings {
		if posting.WordHash == word {
			urls[posting.URLHash] = struct{}{}
		}
	}

	return urls
}

func (o WordPartitionOffer) holdersFrom(
	answers map[yacymodel.Hash]PeerAnswer,
	acceptedURLsPerPeer map[yacymodel.Hash]acceptedURLs,
) []yacymodel.Hash {
	holders := make([]yacymodel.Hash, 0, len(o.Holders)+len(o.Recipients))
	for _, holder := range o.Holders {
		if !slices.Contains(o.StaleHolders, holder) &&
			!o.lostReplicaOn(holder, answers, acceptedURLsPerPeer) {
			holders = append(holders, holder)
		}
	}
	for _, recipient := range o.Recipients {
		if !slices.Contains(o.Holders, recipient.Peer.Hash) &&
			acceptedURLsPerPeer[recipient.Peer.Hash].holdEvery(recipient.Postings) {
			holders = append(holders, recipient.Peer.Hash)
		}
	}

	return holders
}

func (o WordPartitionOffer) lostReplicaOn(
	holder yacymodel.Hash,
	answers map[yacymodel.Hash]PeerAnswer,
	acceptedURLsPerPeer map[yacymodel.Hash]acceptedURLs,
) bool {
	if !o.IsRefresh || !answers[holder].Accepted {
		return false
	}
	for _, recipient := range o.Recipients {
		if recipient.Peer.Hash == holder {
			return !acceptedURLsPerPeer[holder].holdEvery(recipient.Postings)
		}
	}

	return false
}

func (urls acceptedURLs) holdEvery(postings []yacymodel.RWIPosting) bool {
	return !slices.ContainsFunc(postings, func(posting yacymodel.RWIPosting) bool {
		return !urls.hold(posting.URLHash)
	})
}

func (urls acceptedURLs) hold(url yacymodel.URLHash) bool {
	_, found := urls[url]

	return found
}

func sameHolders(holders, otherHolders []yacymodel.Hash) bool {
	return len(holders) == len(otherHolders) &&
		!slices.ContainsFunc(holders, func(holder yacymodel.Hash) bool {
			return !slices.Contains(otherHolders, holder)
		})
}

func (o WordPartitionOffer) deliveredPostingsFrom(
	acceptedURLsPerPeer map[yacymodel.Hash]acceptedURLs,
) []undeliveredpostings.UndeliveredPosting {
	var deliveredPostings []undeliveredpostings.UndeliveredPosting
	for _, posting := range o.UndeliveredPostings {
		if !slices.ContainsFunc(o.ResponsiblePeers, func(peer yacymodel.Hash) bool {
			return !acceptedURLsPerPeer[peer].hold(posting.URL)
		}) {
			deliveredPostings = append(deliveredPostings, posting)
		}
	}

	return deliveredPostings
}

func (o WordPartitionOffer) acceptancesFrom(
	acceptedURLsPerPeer map[yacymodel.Hash]acceptedURLs,
) []Acceptance {
	var acceptances []Acceptance
	for _, recipient := range o.Recipients {
		amountOfPostings := 0
		for _, posting := range recipient.Postings {
			if acceptedURLsPerPeer[recipient.Peer.Hash].hold(posting.URLHash) {
				amountOfPostings++
			}
		}
		if amountOfPostings > 0 {
			acceptances = append(
				acceptances,
				Acceptance{Peer: recipient.Peer.Hash, AmountOfPostings: amountOfPostings},
			)
		}
	}

	return acceptances
}
