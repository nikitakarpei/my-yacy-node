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
	id    nodeidentity.Identity
	base  yacymodel.Seed
	now   func() time.Time
	vault *vault.Vault
	rwi   RWICounter
	urls  URLCounter
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
	seed.Uptime = time.Duration(r.id.Uptime(now)) * time.Minute
	seed.UTCOffset = yacymodel.Some(yacymodel.UTCOffsetOf(now))
	seed.LastSeen = yacymodel.Some(now.UTC())
	seed.IndexedWords = r.countOrZero(ctx, r.rwi.RWICount)
	seed.StoredURLs = r.countOrZero(ctx, r.urls.Count)

	return seed
}

func baseSeed(id nodeidentity.Identity) yacymodel.Seed {
	seed := yacymodel.Seed{
		Hash:         id.Hash,
		Name:         id.Name,
		Port:         yacymodel.Some(yacymodel.Port(id.Port)),
		Capabilities: yacymodel.Some(id.Flags),
		PeerType:     yacymodel.PeerSenior,
		Tags:         yacymodel.MatchAllTags(),
		Version:      yacymodel.Some(id.Version),
	}
	if host, err := yacymodel.ParseHost(id.Host); err == nil {
		seed.PrimaryAddress = yacymodel.Some(host)
	}

	return seed
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
