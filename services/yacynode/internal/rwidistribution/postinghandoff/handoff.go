// Package postinghandoff decides which postings this node may stop holding,
// and deletes them. A posting may go once at least the redundancy in peers
// strictly closer to its DHT position than this node hold a replica of it.
package postinghandoff

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type Reachability interface {
	IsReachable(ctx context.Context, peer yacymodel.Hash) bool
}

type Handoff struct {
	replicas     *postingreplicas.Replicas
	purger       rwipostings.PostingPurger
	reachability Reachability
	partitions   yacymodel.DHTRingPartitions
	self         yacymodel.Hash
	redundancy   int
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func New(
	replicas *postingreplicas.Replicas,
	purger rwipostings.PostingPurger,
	reachability Reachability,
	partitions yacymodel.DHTRingPartitions,
	self yacymodel.Hash,
	redundancy int,
) *Handoff {
	return &Handoff{
		replicas:     replicas,
		purger:       purger,
		reachability: reachability,
		partitions:   partitions,
		self:         self,
		redundancy:   redundancy,
	}
}

func (h *Handoff) HandOffPostingsHeldByCloserPeers(
	ctx context.Context,
	tx *vault.Txn,
	postings []yacymodel.RWIPosting,
) (int, error) {
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
	identity := postingidentity.IdentityOf(posting.WordHash, posting.URLHash)
	holders, err := h.replicas.HoldersOf(tx, identity)
	if err != nil {
		return false, fmt.Errorf("read replica ledger: %w", err)
	}

	position := yacymodel.DHTRingPositionOfPosting(posting, h.partitions)

	return len(h.holdersCloserThanThisNode(ctx, holders, position)) >= h.redundancy, nil
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
		if yacymodel.CloserToDHTRingPosition(peer, h.self, position) {
			closerHolders = append(closerHolders, peer)
		}
	}

	return closerHolders
}
