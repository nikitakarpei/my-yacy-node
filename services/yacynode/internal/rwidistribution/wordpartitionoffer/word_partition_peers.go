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
	holdersAmongResponsiblePeers []yacymodel.Hash,
) wordPartitionPeers {
	return wordPartitionPeers{
		wordPartition:    wordPartition,
		responsiblePeers: responsiblePeers,
		recordedHolders:  recordedHolders,
		staleHolders:     staleHoldersFrom(recordedHolders, holdersAmongResponsiblePeers),
		holdingPeers: holdingPeersIn(
			acceptingPeers.SeedsClockwiseFrom(responsiblePeers.position),
			holdersAmongResponsiblePeers,
		),
		missingHolders: missingHoldersIn(
			acceptingPeers.SeedsClockwiseFrom(responsiblePeers.position),
			recordedHolders,
			responsiblePeers.requiredReplicas-len(holdersAmongResponsiblePeers),
		),
	}
}

func staleHoldersFrom(
	recordedHolders []yacymodel.Hash,
	holdersAmongResponsiblePeers []yacymodel.Hash,
) []yacymodel.Hash {
	var staleHolders []yacymodel.Hash
	for _, holder := range recordedHolders {
		if !slices.Contains(holdersAmongResponsiblePeers, holder) {
			staleHolders = append(staleHolders, holder)
		}
	}

	return staleHolders
}

func holdingPeersIn(
	ringPeers iter.Seq[yacymodel.Seed],
	holdersAmongResponsiblePeers []yacymodel.Hash,
) []yacymodel.Seed {
	var holdingPeers []yacymodel.Seed
	for seed := range ringPeers {
		if slices.Contains(holdersAmongResponsiblePeers, seed.Hash) {
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

func (p wordPartitionPeers) waitReasonFrom(eligibility PeerEligibility) (WaitReason, bool) {
	recipients := p.recipients()
	if len(recipients) == 0 {
		return WaitNoRecipient, true
	}
	if slices.ContainsFunc(recipients, func(seed yacymodel.Seed) bool {
		return eligibility.IsInCooldown(seed.Hash)
	}) {
		return WaitRecipientInCooldown, true
	}

	return "", false
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
	undelivered []undeliveredpostings.UndeliveredPosting,
	undeliveredValues []yacymodel.RWIPosting,
	wholeWordPartition []yacymodel.RWIPosting,
) WordPartitionOffer {
	offer := p.offerWith(undeliveredValues, wholeWordPartition)
	offer.UndeliveredPostings = undelivered

	return offer
}

func (p wordPartitionPeers) offerWith(
	holdingPeerPostings []yacymodel.RWIPosting,
	missingHolderPostings []yacymodel.RWIPosting,
) WordPartitionOffer {
	recipients := make([]Recipient, 0, len(p.holdingPeers)+len(p.missingHolders))
	for _, seed := range p.holdingPeers {
		recipients = append(recipients, Recipient{Peer: seed, Postings: holdingPeerPostings})
	}
	for _, seed := range p.missingHolders {
		recipients = append(recipients, Recipient{Peer: seed, Postings: missingHolderPostings})
	}

	return WordPartitionOffer{
		WordPartition:    p.wordPartition,
		Position:         p.responsiblePeers.position,
		ResponsiblePeers: p.responsiblePeers.hashes,
		Holders:          p.recordedHolders,
		StaleHolders:     p.staleHolders,
		Recipients:       recipients,
	}
}

func (p wordPartitionPeers) refreshOfferWith(
	undelivered []undeliveredpostings.UndeliveredPosting,
	wholeWordPartition []yacymodel.RWIPosting,
) WordPartitionOffer {
	offer := p.offerWith(wholeWordPartition, wholeWordPartition)
	offer.UndeliveredPostings = undelivered
	offer.isRefresh = true

	return offer
}
