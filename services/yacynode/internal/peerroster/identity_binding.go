package peerroster

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type SeedDropReason string

const (
	DropOfSeedWithoutBoundIdentity SeedDropReason = "seed without the bound identity"
	DropOfOlderAddressSignature    SeedDropReason = "older address signature"
)

func dropReasonBetween(
	bound yacymodel.Optional[yacymodel.SignedIdentity],
	next yacymodel.Optional[yacymodel.SignedIdentity],
) yacymodel.Optional[SeedDropReason] {
	current, isBound := bound.Get()
	replacement, verified := next.Get()
	switch {
	case !isBound:
		return yacymodel.None[SeedDropReason]()
	case !verified:
		return yacymodel.Some(DropOfSeedWithoutBoundIdentity)
	default:
		return dropReasonFor(current, replacement)
	}
}

func dropReasonFor(
	bound yacymodel.SignedIdentity,
	replacement yacymodel.SignedIdentity,
) yacymodel.Optional[SeedDropReason] {
	switch {
	case replacement.Identity != bound.Identity:
		return yacymodel.Some(DropOfSeedWithoutBoundIdentity)
	case bound.IssuedAt.After(replacement.IssuedAt):
		return yacymodel.Some(DropOfOlderAddressSignature)
	default:
		return yacymodel.None[SeedDropReason]()
	}
}
