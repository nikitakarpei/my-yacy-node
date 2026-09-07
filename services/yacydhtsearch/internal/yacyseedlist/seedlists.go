// Package yacyseedlist reads the seeds the configured seedlists publish.
package yacyseedlist

import (
	"context"
	"io"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type SeedlistObserver interface {
	SeedlistRead(ctx context.Context, address string, seeds int)
	SeedlistUnreachable(ctx context.Context, address string, cause error)
	SeedlistUnreadable(ctx context.Context, address string, cause error)
}

type Seedlists struct {
	client           *http.Client
	addresses        []string
	maxResponseBytes int64
	observer         SeedlistObserver
}

func New(
	client *http.Client,
	addresses []string,
	maxResponseBytes int64,
	observer SeedlistObserver,
) Seedlists {
	return Seedlists{
		client:           client,
		addresses:        addresses,
		maxResponseBytes: maxResponseBytes,
		observer:         observer,
	}
}

func (s Seedlists) Fetch(ctx context.Context) []yacymodel.Seed {
	seeds := make([]yacymodel.Seed, 0, len(s.addresses))
	for _, address := range s.addresses {
		seeds = append(seeds, s.fetchOne(ctx, address)...)
	}

	return seeds
}

func (s Seedlists) fetchOne(ctx context.Context, address string) []yacymodel.Seed {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		s.observer.SeedlistUnreachable(ctx, address, err)
		return nil
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.observer.SeedlistUnreachable(ctx, address, err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		s.observer.SeedlistUnreadable(ctx, address, statusError(resp.StatusCode))
		return nil
	}

	list, err := yacyproto.ParseSeedListResponse(ctx, io.LimitReader(resp.Body, s.maxResponseBytes))
	if err != nil {
		s.observer.SeedlistUnreadable(ctx, address, err)
		return nil
	}
	s.observer.SeedlistRead(ctx, address, len(list.Seeds))

	return list.Seeds
}
