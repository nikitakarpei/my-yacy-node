// Package postinghandoff decides which postings this node may stop holding,
// and deletes them. A posting may go once at least the redundancy in peers
// strictly closer to its DHT position than this node hold a replica of it.
package postinghandoff

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type Reachability interface {
	Reachable(ctx context.Context, peer yacymodel.Hash) bool
}

type Handoff struct {
	replicas     *postingreplicas.Replicas
	reachability Reachability
	purger       rwipostings.PostingPurger
	partitions   yacymodel.DHTRingPartitions
	self         yacymodel.Hash
	redundancy   int
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func New(
	replicas *postingreplicas.Replicas,
	reachability Reachability,
	purger rwipostings.PostingPurger,
	partitions yacymodel.DHTRingPartitions,
	self yacymodel.Hash,
	redundancy int,
) *Handoff {
	return &Handoff{
		replicas:     replicas,
		reachability: reachability,
		purger:       purger,
		partitions:   partitions,
		self:         self,
		redundancy:   redundancy,
	}
}

// PostingsHeldByCloserPeers reports the postings that enough closer peers hold.
func (h *Handoff) PostingsHeldByCloserPeers(
	ctx context.Context,
	offeredPostings []postingschedule.Identity,
	acceptedRecipients map[postingschedule.Identity][]yacymodel.Hash,
) ([]postingschedule.Identity, error) {
	var held []postingschedule.Identity
	for _, posting := range offeredPostings {
		holders, err := h.holdersOf(ctx, posting, acceptedRecipients[posting])
		if err != nil {
			return nil, err
		}
		if h.closerHolders(posting, holders) >= h.redundancy {
			held = append(held, posting)
		}
	}

	return held, nil
}

func (h *Handoff) HandOffPostings(
	ctx context.Context,
	tx *vault.Txn,
	heldByCloserPeers []postingschedule.Identity,
) error {
	for _, posting := range heldByCloserPeers {
		if _, err := h.purger.PurgePosting(tx, posting.Word, posting.URL); err != nil {
			return err
		}
		slog.DebugContext(ctx, "posting handed off to closer peers",
			slog.String("word", posting.Word.String()),
			slog.String("url", posting.URL.String()))
	}

	return nil
}

func (h *Handoff) holdersOf(
	ctx context.Context,
	posting postingschedule.Identity,
	recipients []yacymodel.Hash,
) ([]yacymodel.Hash, error) {
	recorded, err := h.replicas.Holders(ctx, posting.Word, posting.URL)
	if err != nil {
		return nil, fmt.Errorf("read replica holders: %w", err)
	}

	holders := make([]yacymodel.Hash, 0, len(recorded)+len(recipients))
	for _, peer := range recorded {
		if h.reachability.Reachable(ctx, peer) {
			holders = append(holders, peer)
		}
	}
	for _, peer := range recipients {
		if !slices.Contains(holders, peer) {
			holders = append(holders, peer)
		}
	}

	return holders, nil
}

func (h *Handoff) closerHolders(
	posting postingschedule.Identity,
	holders []yacymodel.Hash,
) int {
	position := yacymodel.PostingPosition(posting.Word, posting.URL, h.partitions)

	var closer int
	for _, peer := range holders {
		if yacymodel.CloserToPosition(peer, h.self, position) {
			closer++
		}
	}

	return closer
}
