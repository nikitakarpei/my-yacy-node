package wordpartitionoffer

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
)

type Offer struct {
	Recipients          []Recipient
	wordPartition       yacymodel.WordPartition
	position            yacymodel.DHTRingPosition
	responsiblePeers    []yacymodel.Hash
	recordedHolders     []yacymodel.Hash
	staleHolders        []yacymodel.Hash
	undeliveredPostings []undeliveredpostings.Posting
	isRefresh           bool
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
	WordPartition        yacymodel.WordPartition
	Position             yacymodel.DHTRingPosition
	ResponsiblePeers     []yacymodel.Hash
	Holders              []yacymodel.Hash
	IsAtRedundancy       bool
	IsRefresh            bool
	DeliveredPostings    []undeliveredpostings.Posting
	AmountOfStaleHolders int
	Acceptances          []Acceptance
}

type Acceptance struct {
	Peer             yacymodel.Hash
	AmountOfPostings int
}

type acceptedURLs map[yacymodel.URLHash]struct{}

func (o Offer) OutcomeFrom(answers map[yacymodel.Hash]PeerAnswer) Outcome {
	acceptedURLsPerPeer := o.acceptedURLsPerPeerFrom(answers)
	holders := o.holdersFrom(answers, acceptedURLsPerPeer)
	deliveredPostings := o.deliveredPostingsFrom(acceptedURLsPerPeer)
	isAtRedundancy := o.isAtRedundancyWith(holders, deliveredPostings)

	return Outcome{
		WordPartition:        o.wordPartition,
		Position:             o.position,
		ResponsiblePeers:     o.responsiblePeers,
		Holders:              holders,
		IsAtRedundancy:       isAtRedundancy,
		IsRefresh:            o.isRefresh,
		DeliveredPostings:    deliveredPostings,
		AmountOfStaleHolders: len(o.staleHolders),
		Acceptances:          o.acceptancesFrom(acceptedURLsPerPeer),
	}
}

func (o Offer) acceptedURLsPerPeerFrom(
	answers map[yacymodel.Hash]PeerAnswer,
) map[yacymodel.Hash]acceptedURLs {
	acceptedURLsPerPeer := make(map[yacymodel.Hash]acceptedURLs)
	for peer, answer := range answers {
		if o.isRecipient(peer) || slices.Contains(o.responsiblePeers, peer) {
			acceptedURLsPerPeer[peer] = acceptedURLsOfWord(answer, o.wordPartition.Word)
		}
	}

	return acceptedURLsPerPeer
}

func (o Offer) isRecipient(peer yacymodel.Hash) bool {
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

func (o Offer) holdersFrom(
	answers map[yacymodel.Hash]PeerAnswer,
	acceptedURLsPerPeer map[yacymodel.Hash]acceptedURLs,
) []yacymodel.Hash {
	holders := make([]yacymodel.Hash, 0, len(o.recordedHolders)+len(o.Recipients))
	for _, holder := range o.recordedHolders {
		if !slices.Contains(o.staleHolders, holder) &&
			!o.isLostHolder(holder, answers, acceptedURLsPerPeer) {
			holders = append(holders, holder)
		}
	}
	for _, recipient := range o.Recipients {
		if !slices.Contains(o.recordedHolders, recipient.Peer.Hash) &&
			acceptedURLsPerPeer[recipient.Peer.Hash].holdEvery(recipient.Postings) {
			holders = append(holders, recipient.Peer.Hash)
		}
	}

	return holders
}

func (o Offer) isLostHolder(
	holder yacymodel.Hash,
	answers map[yacymodel.Hash]PeerAnswer,
	acceptedURLsPerPeer map[yacymodel.Hash]acceptedURLs,
) bool {
	if !o.isRefresh || !answers[holder].Accepted {
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

func (o Offer) deliveredPostingsFrom(
	acceptedURLsPerPeer map[yacymodel.Hash]acceptedURLs,
) []undeliveredpostings.Posting {
	var deliveredPostings []undeliveredpostings.Posting
	for _, posting := range o.undeliveredPostings {
		if !slices.ContainsFunc(o.responsiblePeers, func(peer yacymodel.Hash) bool {
			return !acceptedURLsPerPeer[peer].hold(posting.URL)
		}) {
			deliveredPostings = append(deliveredPostings, posting)
		}
	}

	return deliveredPostings
}

func (o Offer) isAtRedundancyWith(
	holders []yacymodel.Hash,
	deliveredPostings []undeliveredpostings.Posting,
) bool {
	return len(holders) >= len(o.responsiblePeers) &&
		len(deliveredPostings) == len(o.undeliveredPostings)
}

func (o Offer) acceptancesFrom(
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
