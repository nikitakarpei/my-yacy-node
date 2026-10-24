package nodestatus

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
)

const msgCountUnavailable = "count unavailable for self seed"

type runtimeStatus struct {
	id            nodeidentity.Identity
	addressSource PublicAddressSource
	base          yacymodel.Seed
	now           func() time.Time
	vault         *vault.Vault
	words         WordAmountEstimate
	urls          URLCounter
}

func (r runtimeStatus) Version(context.Context) string {
	return strconv.FormatFloat(r.id.Version.Release, 'f', -1, 64)
}

func (r runtimeStatus) NetworkName(context.Context) string {
	return r.id.NetworkName
}

func (r runtimeStatus) Uptime(context.Context) int {
	return r.id.Uptime(r.now())
}

func (r runtimeStatus) SelfSeed(ctx context.Context) yacymodel.Seed {
	now := r.now()
	seed := r.base
	seed.PrimaryAddress = r.addressSource.Host(ctx)
	seed.Uptime = time.Duration(r.id.Uptime(now)) * time.Minute
	seed.UTCOffset = yacymodel.Some(yacymodel.UTCOffsetOf(now))
	seed.LastSeen = yacymodel.Some(now.UTC())
	seed.IndexedWords = r.countOrZero(ctx, r.words.EstimatedAmountOfWords)
	seed.StoredURLs = r.countOrZero(ctx, r.urls.Count)

	return seed
}

func baseSeed(id nodeidentity.Identity) yacymodel.Seed {
	return yacymodel.Seed{
		Hash:         id.Hash,
		Name:         id.Name,
		Port:         yacymodel.Some(yacymodel.Port(id.Port)),
		Capabilities: yacymodel.Some(id.Capabilities),
		PeerType:     yacymodel.PeerSenior,
		Tags:         yacymodel.MatchAllTags(),
		Version:      yacymodel.Some(id.Version),
	}
}

func (r runtimeStatus) countOrZero(ctx context.Context, count func(*vault.Txn) (int, error)) int {
	var stored int
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		measured, err := count(tx)
		stored = measured

		return err
	}); err != nil {
		slog.WarnContext(ctx, msgCountUnavailable, slog.Any("error", err))

		return 0
	}

	return stored
}
