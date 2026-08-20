package ordertraversal

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const msgSeedRejected = "seed url rejected"

func canonicalSeeds(ctx context.Context, seedURLs []string) []string {
	seeds := make([]string, 0, len(seedURLs))
	for _, seed := range seedURLs {
		canonical, err := yacycrawlcontract.CanonicalURLOf(seed)
		if err != nil {
			slog.WarnContext(ctx, msgSeedRejected,
				slog.String("url", seed),
				slog.Any("error", err),
			)
			continue
		}
		seeds = append(seeds, canonical)
	}
	return seeds
}
