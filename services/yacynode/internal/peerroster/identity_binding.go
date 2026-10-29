package peerroster

import (
	"maps"
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
	staleSeed
	replayedSeed
	foreignSeed
	futureSeed
)

func standingOf(offered, known rosterEntry, now time.Time) seedStanding {
	next, verified := offered.binding.Get()
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
	case sameAddresses(offered.addresses, known.addresses):
		return staleSeed
	default:
		return replayedSeed
	}
}

func sameAddresses(first, second yacymodel.PeerAddresses) bool {
	return first.Port == second.Port &&
		first.SecurePort == second.SecurePort &&
		maps.Equal(hostSetOf(first.Hosts), hostSetOf(second.Hosts))
}

func hostSetOf(hosts []yacymodel.Host) map[yacymodel.Host]struct{} {
	hostSet := make(map[yacymodel.Host]struct{}, len(hosts))
	for _, host := range hosts {
		hostSet[host] = struct{}{}
	}

	return hostSet
}
