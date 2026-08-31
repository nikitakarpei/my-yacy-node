package yacyproto

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const seedListMaxLineBytes = 1 << 20

type SeedListResponse struct {
	Seeds []yacymodel.Seed
}

func ParseSeedListResponse(
	ctx context.Context,
	seedList io.Reader,
) (SeedListResponse, error) {
	var seeds []yacymodel.Seed
	seedLines := bufio.NewScanner(seedList)
	seedLines.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), seedListMaxLineBytes)
	for seedLineNumber := 1; seedLines.Scan(); seedLineNumber++ {
		seedLine := seedLines.Text()
		if seedLine == "" {
			continue
		}

		seed, err := seedWireCodec{}.decodeRemote(ctx, seedLine)
		if err != nil {
			slog.WarnContext(
				ctx,
				"seed list line discarded",
				slog.Int("line", seedLineNumber),
				slog.Any("error", err),
			)

			continue
		}

		seeds = append(seeds, seed)
	}
	if err := seedLines.Err(); err != nil {
		return SeedListResponse{}, fmt.Errorf("read seed list: %w", err)
	}

	return SeedListResponse{Seeds: seeds}, nil
}
