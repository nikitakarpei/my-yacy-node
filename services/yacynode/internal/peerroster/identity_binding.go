package peerroster

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const issuedAtTolerance = 10 * time.Minute

type SeedDropReason string

const (
	DropOfSeedWithoutBoundIdentity SeedDropReason = "seed without the bound identity"
	DropOfOlderAddressSignature    SeedDropReason = "older address signature"
	DropOfFutureAddressSignature   SeedDropReason = "future address signature"
)

type seedStanding int

const (
	legacySeed seedStanding = iota
	newerSeed
	currentSeed
	olderSeed
	foreignSeed
	futureSeed
)

func dropReasonOf(standing seedStanding) yacymodel.Optional[SeedDropReason] {
	switch standing {
	case olderSeed:
		return yacymodel.Some(DropOfOlderAddressSignature)
	case foreignSeed:
		return yacymodel.Some(DropOfSeedWithoutBoundIdentity)
	case futureSeed:
		return yacymodel.Some(DropOfFutureAddressSignature)
	case legacySeed, newerSeed, currentSeed:
	}

	return yacymodel.None[SeedDropReason]()
}

func standingOf(
	verifiedSignature yacymodel.Optional[yacymodel.VerifiedAddressSignature],
	known rosterEntry,
	now time.Time,
) seedStanding {
	next, verified := verifiedSignature.Get()
	bound, isBound := known.binding.Get()
	switch {
	case verified && next.IssuedAt.After(now.Add(issuedAtTolerance)):
		return futureSeed
	case !isBound && verified:
		return newerSeed
	case !isBound:
		return legacySeed
	case !verified || next.Identity != bound.Identity:
		return foreignSeed
	case next.IssuedAt.After(bound.IssuedAt):
		return newerSeed
	case next.IssuedAt.Equal(bound.IssuedAt):
		return currentSeed
	default:
		return olderSeed
	}
}
