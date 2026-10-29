// Package peerroster owns the set of network peers this node knows. It is the
// single owner of each peer's recency and reachable membership: the announcement
// loop maintains the roster from contact outcomes, while inbound admission samples
// and refreshes it. Only the bounded reachable set lives in memory; every known peer
// is persisted, so a restart resumes from the durable roster instead of the seed
// source. A peer stays credible for a bounded number of announce rounds after its
// last confirmed reachable contact, so a cold reachable set does not read as an
// empty one.
package peerroster

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

type Roster interface {
	Discover(ctx context.Context, seeds ...yacymodel.Seed)
	ConfirmFirsthandSeed(ctx context.Context, seed yacymodel.Seed, answeredBaseURL peerwire.BaseURL)
	ConfirmSecondhandSeed(
		ctx context.Context,
		seed yacymodel.Seed,
		answeredBaseURL peerwire.BaseURL,
	)
	ConfirmUnreachable(ctx context.Context, peer yacymodel.Hash)
	ReachablePeers(ctx context.Context) []yacymodel.Seed
	MostRecentlyReachablePeers(ctx context.Context, limit int) []yacymodel.Seed
	IsReachable(ctx context.Context, peer yacymodel.Hash) bool
	IsRecentlyReachable(ctx context.Context, peer yacymodel.Hash) bool
	UnreachablePeerHashes(ctx context.Context, limit int) []yacymodel.Hash
	RecordOf(ctx context.Context, peer yacymodel.Hash) (PeerRecord, bool)
}

var _ Roster = (*roster)(nil)

//nolint:revive // argument-limit: eight explicit, independently-meaningful collaborators
func Open(
	storage *vault.Vault,
	now func() time.Time,
	reservoirCap int,
	reachableCap int,
	announceInterval time.Duration,
	self yacymodel.Hash,
	networkName string,
	observer RosterObserver,
) (Roster, error) {
	peers, err := registerRoster(storage)
	if err != nil {
		return nil, err
	}

	return &roster{
		vault:            storage,
		peers:            peers,
		now:              now,
		reservoirCap:     reservoirCap,
		announceInterval: announceInterval,
		self:             self,
		networkName:      networkName,
		observer:         observer,
		reachable:        newReachablePeers(reachableCap),
	}, nil
}
