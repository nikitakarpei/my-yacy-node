// Package postinghandoff decides which postings this node may stop holding,
// and deletes them. A posting may go once at least the redundancy in peers
// strictly closer to its DHT position than this node hold a replica of it. A
// node that keeps every posting hands off none.
package postinghandoff

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type Reachability interface {
	IsReachable(ctx context.Context, peer yacymodel.Hash) bool
}

type Config struct {
	Self       yacymodel.Hash
	Redundancy int
}

type Handoff struct {
	purger       rwipostings.PostingPurger
	reachability Reachability
	config       Config
}

func New(
	purger rwipostings.PostingPurger,
	reachability Reachability,
	config Config,
) *Handoff {
	return &Handoff{
		purger:       purger,
		reachability: reachability,
		config:       config,
	}
}

func (h *Handoff) HandOffIfHeldByCloserPeers(
	ctx context.Context,
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	position yacymodel.DHTRingPosition,
	holders []yacymodel.Hash,
) (bool, error) {
	if len(h.holdersCloserThanThisNode(ctx, holders, position)) < h.config.Redundancy {
		return false, nil
	}

	wasPurged, err := h.purger.PurgePosting(tx, posting.WordHash, posting.URLHash)
	if err != nil || !wasPurged {
		return false, err
	}
	slog.DebugContext(ctx, "posting handed off to closer peers",
		slog.String("word", posting.WordHash.String()),
		slog.String("url", posting.URLHash.String()))

	return true, nil
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
