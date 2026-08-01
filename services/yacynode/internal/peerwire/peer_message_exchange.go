// Package peerwire posts a form-encoded request to a peer endpoint and returns
// the reply as a wire message. It reports what the peer answered on a failure,
// including the status line and the start of the reported body.
package peerwire

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

const responseMaxBodyBytes int64 = 1 << 20

var errPeerExchange = errors.New("peer exchange failed")

type MessageExchange struct {
	client *http.Client
}

func NewMessageExchange(client *http.Client) MessageExchange {
	return MessageExchange{client: client}
}

func (e MessageExchange) Exchange(
	ctx context.Context,
	endpoint, path string,
	form url.Values,
) (yacyproto.Message, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("%w: empty endpoint", errPeerExchange)
	}
	target := url.URL{Scheme: "http", Host: endpoint, Path: path}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target.String(),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errPeerExchange, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errPeerExchange, err)
	}
	defer closeResponseBody(ctx, resp.Body, path)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: status %s: %s",
			errPeerExchange,
			resp.Status,
			peerFailureReport(resp.Body),
		)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, responseMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errPeerExchange, err)
	}

	return yacyproto.ParseMessage(string(raw)), nil
}
