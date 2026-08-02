package distributioncycle

import (
	"context"
	"log/slog"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicashortfall"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type postingHandoff struct {
	purger rwipostings.PostingPurger
}

func newPostingHandoff(purger rwipostings.PostingPurger) *postingHandoff {
	return &postingHandoff{purger: purger}
}

func (h *postingHandoff) HandOffPostings(
	ctx context.Context,
	tx *vault.Txn,
	handoffs []replicashortfall.ReplicaHandoff,
	acceptedRecipients map[postingschedule.Identity][]yacymodel.Hash,
) ([]postingschedule.Identity, error) {
	var handedOff []postingschedule.Identity
	for _, handoff := range handoffs {
		if !closerPeersHold(handoff, acceptedRecipients[handoff.Posting]) {
			continue
		}
		if err := h.handOffPosting(ctx, tx, handoff.Posting); err != nil {
			return nil, err
		}
		handedOff = append(handedOff, handoff.Posting)
	}

	return handedOff, nil
}

func (h *postingHandoff) handOffPosting(
	ctx context.Context,
	tx *vault.Txn,
	identity postingschedule.Identity,
) error {
	if _, err := h.purger.PurgePosting(tx, identity.Word, identity.URL); err != nil {
		return err
	}
	slog.DebugContext(ctx, "posting handed off to closer peers",
		slog.String("word", identity.Word.String()),
		slog.String("url", identity.URL.String()))

	return nil
}

func closerPeersHold(
	handoff replicashortfall.ReplicaHandoff,
	recipients []yacymodel.Hash,
) bool {
	var closer int
	for _, recipient := range recipients {
		if slices.Contains(handoff.CloserPeersOffered, recipient) {
			closer++
		}
	}

	return closer >= handoff.CloserPeersNeeded
}
