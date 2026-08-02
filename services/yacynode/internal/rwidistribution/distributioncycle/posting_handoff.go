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
	replicaOffers []replicashortfall.ReplicaOffer,
	acceptedRecipients map[postingschedule.Identity][]yacymodel.Hash,
) ([]replicashortfall.ReplicaOffer, error) {
	kept := make([]replicashortfall.ReplicaOffer, 0, len(replicaOffers))
	for _, replicaOffer := range replicaOffers {
		identity := postingschedule.Identity{
			Word: replicaOffer.Posting.WordHash,
			URL:  replicaOffer.Posting.URLHash,
		}
		if !closerPeersHold(replicaOffer, acceptedRecipients[identity]) {
			kept = append(kept, replicaOffer)

			continue
		}
		if err := h.handOffPosting(ctx, tx, identity); err != nil {
			return nil, err
		}
	}

	return kept, nil
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
	replicaOffer replicashortfall.ReplicaOffer,
	recipients []yacymodel.Hash,
) bool {
	var closer int
	for _, recipient := range recipients {
		if slices.Contains(replicaOffer.RecipientsCloserThanThisNode, recipient) {
			closer++
		}
	}

	return closer >= replicaOffer.HandoffReplicasNeeded
}
