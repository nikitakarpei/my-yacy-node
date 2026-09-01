package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const seedlistMaxBodyBytes int64 = 8 << 20

var errSeedlistFetchFailed = errors.New("seedlist fetch failed")

type httpSeedlistFetcher struct {
	client *http.Client
}

func newHTTPSeedlistFetcher(client *http.Client) httpSeedlistFetcher {
	return httpSeedlistFetcher{client: client}
}

func (f httpSeedlistFetcher) Fetch(ctx context.Context, url string) ([]yacymodel.Seed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSeedlistFetchFailed, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSeedlistFetchFailed, err)
	}
	defer closeResponseBody(ctx, resp.Body, "seedlistFetch")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", errSeedlistFetchFailed, resp.StatusCode)
	}

	seedListResponse, err := yacyproto.ParseSeedListResponse(
		ctx,
		io.LimitReader(resp.Body, seedlistMaxBodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSeedlistFetchFailed, err)
	}

	return seedListResponse.Seeds, nil
}
