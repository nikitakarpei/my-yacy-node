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
	unprovenSeed
	futureSeed
)

func dropReasonOf(standing seedStanding) yacymodel.Optional[SeedDropReason] {
	switch standing {
	case olderSeed:
		return yacymodel.Some(DropOfOlderAddressSignature)
	case unprovenSeed:
		return yacymodel.Some(DropOfSeedWithoutBoundIdentity)
	case futureSeed:
		return yacymodel.Some(DropOfFutureAddressSignature)
	case legacySeed, newerSeed, currentSeed:
	}

	return yacymodel.None[SeedDropReason]()
}

func (s seedStanding) replacesSeed() bool {
	return s == legacySeed || s == newerSeed
}

func seedStandingAgainst(
	boundSignature yacymodel.Optional[yacymodel.VerifiedAddressSignature],
	presentedSignature yacymodel.Optional[yacymodel.VerifiedAddressSignature],
	now time.Time,
) seedStanding {
	bound, isBound := boundSignature.Get()
	presented, isPresented := presentedSignature.Get()
	switch {
	case isPresented && presented.IssuedAt.After(now.Add(issuedAtTolerance)):
		return futureSeed
	case !isBound && isPresented:
		return newerSeed
	case !isBound:
		return legacySeed
	case !isPresented || presented.Identity != bound.Identity:
		return unprovenSeed
	case presented.IssuedAt.After(bound.IssuedAt):
		return newerSeed
	case presented.IssuedAt.Equal(bound.IssuedAt):
		return currentSeed
	default:
		return olderSeed
	}
}
