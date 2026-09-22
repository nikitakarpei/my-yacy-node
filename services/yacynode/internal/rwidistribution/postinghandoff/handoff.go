// Package postinghandoff decides which postings this node may stop holding,
// and deletes them. A posting may go once at least the redundancy in peers
// strictly closer to its DHT position than this node hold a replica of it. A
// node with handoff disabled keeps every posting.
package postinghandoff

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type Reachability interface {
	IsReachable(ctx context.Context, peer yacymodel.Hash) bool
}

type Config struct {
	Enabled    bool
	Partitions yacymodel.DHTRingPartitions
	Self       yacymodel.Hash
	Redundancy int
}

type Handoff struct {
	replicas     *postingreplicas.Replicas
	purger       rwipostings.PostingPurger
	reachability Reachability
	config       Config
}

func New(
	replicas *postingreplicas.Replicas,
	purger rwipostings.PostingPurger,
	reachability Reachability,
	config Config,
) *Handoff {
	return &Handoff{
		replicas:     replicas,
		purger:       purger,
		reachability: reachability,
		config:       config,
	}
}

func (h *Handoff) HandOffPostingsHeldByCloserPeers(
	ctx context.Context,
	tx *vault.Txn,
	postings []yacymodel.RWIPosting,
) (int, error) {
	if !h.config.Enabled {
		return 0, nil
	}

	var handedOffPostings int
	for _, posting := range postings {
		heldByCloserPeers, err := h.isHeldByCloserPeers(ctx, tx, posting)
		if err != nil {
			return 0, err
		}
		if !heldByCloserPeers {
			continue
		}

		if _, err := h.purger.PurgePosting(
			tx, posting.WordHash, posting.URLHash,
		); err != nil {
			return 0, err
		}
		slog.DebugContext(ctx, "posting handed off to closer peers",
			slog.String("word", posting.WordHash.String()),
			slog.String("url", posting.URLHash.String()))
		handedOffPostings++
	}

	return handedOffPostings, nil
}

func (h *Handoff) isHeldByCloserPeers(
	ctx context.Context,
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
) (bool, error) {
	identity := postingidentity.IdentityOf(posting)
	holders, err := h.replicas.HoldersOf(tx, identity)
	if err != nil {
		return false, fmt.Errorf("read replica ledger: %w", err)
	}

	position := yacymodel.DHTRingPositionOfPosting(posting, h.config.Partitions)

	return len(h.holdersCloserThanThisNode(ctx, holders, position)) >= h.config.Redundancy, nil
}

func (h *Handoff) holdersCloserThanThisNode(
	ctx context.Context,
	holders []yacymodel.Hash,
	position yacymodel.DHTRingPosition,
) []yacymodel.Hash {
	closerHolders := make([]yacymodel.Hash, 0, len(holders))
	for _, peer := range holders {
		if !h.reachability.IsReachable(ctx, peer) {
			continue
		}
		if yacymodel.CloserToDHTRingPosition(peer, h.config.Self, position) {
			closerHolders = append(closerHolders, peer)
		}
	}

	return closerHolders
}
