package wordpartitionoffer

import (
	"iter"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
)

type wordPartitionPeers struct {
	wordPartition    yacymodel.WordPartition
	responsiblePeers responsiblePeers
	recordedHolders  []yacymodel.Hash
	staleHolders     []yacymodel.Hash
	holdingPeers     []yacymodel.Seed
	missingHolders   []yacymodel.Seed
}

func wordPartitionPeersFrom(
	wordPartition yacymodel.WordPartition,
	responsiblePeers responsiblePeers,
	acceptingPeers yacymodel.SeedRing,
	recordedHolders []yacymodel.Hash,
	keptHolders []yacymodel.Hash,
) wordPartitionPeers {
	return wordPartitionPeers{
		wordPartition:    wordPartition,
		responsiblePeers: responsiblePeers,
		recordedHolders:  recordedHolders,
		staleHolders:     staleHoldersFrom(recordedHolders, keptHolders),
		holdingPeers: holdingPeersIn(
			acceptingPeers.SeedsClockwiseFrom(responsiblePeers.position),
			keptHolders,
		),
		missingHolders: missingHoldersIn(
			acceptingPeers.SeedsClockwiseFrom(responsiblePeers.position),
			recordedHolders,
			responsiblePeers.amountOfRequiredHolders-len(keptHolders),
		),
	}
}

func staleHoldersFrom(
	recordedHolders []yacymodel.Hash,
	keptHolders []yacymodel.Hash,
) []yacymodel.Hash {
	var staleHolders []yacymodel.Hash
	for _, holder := range recordedHolders {
		if !slices.Contains(keptHolders, holder) {
			staleHolders = append(staleHolders, holder)
		}
	}

	return staleHolders
}

func holdingPeersIn(
	ringPeers iter.Seq[yacymodel.Seed],
	keptHolders []yacymodel.Hash,
) []yacymodel.Seed {
	var holdingPeers []yacymodel.Seed
	for seed := range ringPeers {
		if slices.Contains(keptHolders, seed.Hash) {
			holdingPeers = append(holdingPeers, seed)
		}
	}

	return holdingPeers
}

func missingHoldersIn(
	ringPeers iter.Seq[yacymodel.Seed],
	recordedHolders []yacymodel.Hash,
	amountOfMissingHolders int,
) []yacymodel.Seed {
	var missingHolders []yacymodel.Seed
	for seed := range ringPeers {
		if len(missingHolders) >= amountOfMissingHolders {
			break
		}
		if !slices.Contains(recordedHolders, seed.Hash) {
			missingHolders = append(missingHolders, seed)
		}
	}

	return missingHolders
}

func (p wordPartitionPeers) mustWaitFor(eligibility PeerEligibility) bool {
	recipients := p.recipients()

	return len(recipients) == 0 ||
		anyRecipient(recipients, eligibility.IsPaused) ||
		anyRecipient(recipients, eligibility.IsBusy)
}

func anyRecipient(recipients []yacymodel.Seed, peerCondition func(peer yacymodel.Hash) bool) bool {
	return slices.ContainsFunc(recipients, func(seed yacymodel.Seed) bool {
		return peerCondition(seed.Hash)
	})
}

func (p wordPartitionPeers) hasHoldingPeers() bool {
	return len(p.holdingPeers) > 0
}

func (p wordPartitionPeers) hasMissingHolders() bool {
	return len(p.missingHolders) > 0
}

func (p wordPartitionPeers) recipients() []yacymodel.Seed {
	return slices.Concat(p.holdingPeers, p.missingHolders)
}

func (p wordPartitionPeers) undeliveredOfferWith(
	undelivered []undeliveredpostings.Posting,
	undeliveredValues []yacymodel.RWIPosting,
	wholeWordPartition []yacymodel.RWIPosting,
) Offer {
	offer := p.offerWith(undeliveredValues, wholeWordPartition)
	offer.undeliveredPostings = undelivered

	return offer
}

func (p wordPartitionPeers) offerWith(
	holdingPeerPostings []yacymodel.RWIPosting,
	missingHolderPostings []yacymodel.RWIPosting,
) Offer {
	recipients := make([]Recipient, 0, len(p.holdingPeers)+len(p.missingHolders))
	for _, seed := range p.holdingPeers {
		recipients = append(recipients, Recipient{Peer: seed, Postings: holdingPeerPostings})
	}
	for _, seed := range p.missingHolders {
		recipients = append(recipients, Recipient{Peer: seed, Postings: missingHolderPostings})
	}

	return Offer{
		Recipients:       recipients,
		wordPartition:    p.wordPartition,
		position:         p.responsiblePeers.position,
		responsiblePeers: p.responsiblePeers.hashes,
		recordedHolders:  p.recordedHolders,
		staleHolders:     p.staleHolders,
		isRefresh:        !p.hasHoldingPeers(),
	}
}

func (p wordPartitionPeers) refreshOfferWith(
	undelivered []undeliveredpostings.Posting,
	wholeWordPartition []yacymodel.RWIPosting,
) Offer {
	offer := p.offerWith(wholeWordPartition, wholeWordPartition)
	offer.undeliveredPostings = undelivered
	offer.isRefresh = true

	return offer
}
