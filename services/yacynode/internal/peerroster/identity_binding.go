package peerroster

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type SeedDropReason string

const (
	DropOfSeedWithoutBoundIdentity SeedDropReason = "seed without the bound identity"
	DropOfOlderAddressSignature    SeedDropReason = "older address signature"
)

type identityBinding struct {
	identity yacymodel.PeerIdentity
	issuedAt time.Time
}

func bindingFrom(seed yacymodel.Seed, networkName string) yacymodel.Optional[identityBinding] {
	identity, verified := seed.VerifiedIdentityIn(networkName)
	if !verified {
		return yacymodel.None[identityBinding]()
	}
	signature, _ := seed.AddressSignature.Get()

	return yacymodel.Some(identityBinding{identity: identity, issuedAt: signature.IssuedAt})
}

func dropReasonBetween(
	bound yacymodel.Optional[identityBinding],
	next yacymodel.Optional[identityBinding],
) yacymodel.Optional[SeedDropReason] {
	current, isBound := bound.Get()
	replacement, verified := next.Get()
	switch {
	case !isBound:
		return yacymodel.None[SeedDropReason]()
	case !verified:
		return yacymodel.Some(DropOfSeedWithoutBoundIdentity)
	default:
		return current.dropReasonFor(replacement)
	}
}

func (b identityBinding) dropReasonFor(
	replacement identityBinding,
) yacymodel.Optional[SeedDropReason] {
	switch {
	case replacement.identity != b.identity:
		return yacymodel.Some(DropOfSeedWithoutBoundIdentity)
	case b.issuedAt.After(replacement.issuedAt):
		return yacymodel.Some(DropOfOlderAddressSignature)
	default:
		return yacymodel.None[SeedDropReason]()
	}
}
