package nodestatus

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const msgCountUnavailable = "count unavailable for self seed"

type runtimeStatus struct {
	id   nodeidentity.Identity
	base yacymodel.Seed
	now  func() time.Time
	rwi  RWICounter
	urls URLCounter
}

func (r runtimeStatus) Version(context.Context) string {
	return r.id.Version
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
	seed.IndexedWords = countOrZero(ctx, r.rwi.RWICount)
	seed.StoredURLs = countOrZero(ctx, r.urls.Count)

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
	}
	if version, err := yacyproto.ParseSoftwareVersion(id.Version); err == nil {
		seed.Version = yacymodel.Some(version)
	}
	if host, err := yacymodel.ParseHost(id.Host); err == nil {
		seed.PrimaryAddress = yacymodel.Some(host)
	}

	return seed
}

func countOrZero(ctx context.Context, fn func(context.Context) (int, error)) int {
	n, err := fn(ctx)
	if err != nil {
		slog.WarnContext(ctx, msgCountUnavailable, slog.Any("error", err))

		return 0
	}

	return n
}
