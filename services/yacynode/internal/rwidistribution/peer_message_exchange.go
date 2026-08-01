package rwidistribution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const peerResponseMaxBodyBytes int64 = 1 << 20

var errPeerRequest = errors.New("peer request failed")

type peerMessageExchange struct {
	client *http.Client
}

func (e peerMessageExchange) Exchange(
	ctx context.Context,
	endpoint, path string,
	form url.Values,
) (yacyproto.Message, error) {
	target := url.URL{Scheme: "http", Host: endpoint, Path: path}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target.String(),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errPeerRequest, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errPeerRequest, err)
	}
	defer closeResponseBody(ctx, resp.Body, path)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", errPeerRequest, resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, peerResponseMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errPeerRequest, err)
	}

	return yacyproto.ParseMessage(string(raw)), nil
}
