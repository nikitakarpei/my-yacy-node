// Package postinghandoff decides which postings this node may stop holding,
// and deletes them. A posting may go once the redundancy in its responsible
// peers, strictly closer to its DHT position than this node, hold a replica of
// it. A node that keeps every posting hands off none.
package postinghandoff

import (
	"context"
	"log/slog"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type Config struct {
	Self       yacymodel.Hash
	Redundancy int
}

type Handoff struct {
	purger rwipostings.PostingPurger
	config Config
}

func New(purger rwipostings.PostingPurger, config Config) *Handoff {
	return &Handoff{
		purger: purger,
		config: config,
	}
}

//nolint:revive // argument-limit: the posting, its DHT position, its holders and its responsible peers each feed the rule
func (h *Handoff) HandOffIfHeldByResponsiblePeers(
	ctx context.Context,
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	position yacymodel.DHTRingPosition,
	holders []yacymodel.Hash,
	responsiblePeers []yacymodel.Hash,
) (bool, error) {
	if len(h.closerResponsibleHolders(holders, responsiblePeers, position)) < h.config.Redundancy {
		return false, nil
	}

	wasPurged, err := h.purger.PurgePosting(tx, posting.WordHash, posting.URLHash)
	if err != nil || !wasPurged {
		return false, err
	}
	slog.DebugContext(ctx, "posting handed off to responsible peers",
		slog.String("word", posting.WordHash.String()),
		slog.String("url", posting.URLHash.String()))

	return true, nil
}

func (h *Handoff) closerResponsibleHolders(
	holders []yacymodel.Hash,
	responsiblePeers []yacymodel.Hash,
	position yacymodel.DHTRingPosition,
) []yacymodel.Hash {
	closerHolders := make([]yacymodel.Hash, 0, len(holders))
	for _, peer := range holders {
		if slices.Contains(responsiblePeers, peer) &&
			yacymodel.CloserToDHTRingPosition(peer, h.config.Self, position) {
			closerHolders = append(closerHolders, peer)
		}
	}

	return closerHolders
}
