package peerwire

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const responseMaxBodyBytes int64 = 1 << 20

const (
	failureReportMaxBytes int64 = 512
	failureReportAbsent         = "no reported body"
)

var (
	errUnexpectedStatus   = errors.New("unexpected status")
	errRequestNotPrepared = errors.New("request not prepared")
)

type roundTripper struct {
	client         *http.Client
	peerTransports PeerTransports
}

func newRoundTripper(client *http.Client, peerTransports PeerTransports) roundTripper {
	peerClient := *client
	peerClient.CheckRedirect = refuseRedirect

	return roundTripper{client: &peerClient, peerTransports: peerTransports}
}

func refuseRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func (p roundTripper) post(
	ctx context.Context,
	identity yacymodel.Optional[yacymodel.PeerIdentity],
	baseURL BaseURL,
	path string,
	body multipartForm,
) (yacyproto.ResponseFields, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL.urlOf(path),
		bytes.NewReader(body.content),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRequestNotPrepared, err)
	}
	req.Header.Set("Content-Type", body.contentType)

	resp, err := p.clientFor(identity, baseURL).Do(req)
	if err != nil {
		return nil, fmt.Errorf("post: %w", err)
	}
	defer closeResponseBody(ctx, resp.Body, path)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"%w %s: %s",
			errUnexpectedStatus,
			resp.Status,
			peerFailureReport(resp.Body),
		)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, responseMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read reply: %w", err)
	}

	return yacyproto.ParseResponseFields(raw), nil
}

func (p roundTripper) clientFor(
	identity yacymodel.Optional[yacymodel.PeerIdentity],
	baseURL BaseURL,
) *http.Client {
	expected, identified := identity.Get()
	if !identified || baseURL.Scheme != HTTPS {
		return p.client
	}
	peerClient := *p.client
	peerClient.Transport = p.peerTransports.TransportFor(expected)

	return &peerClient
}

func closeResponseBody(ctx context.Context, body io.Closer, path string) {
	if err := body.Close(); err != nil {
		slog.WarnContext(
			ctx,
			"response body close failed",
			slog.String("path", path),
			slog.Any("error", err),
		)
	}
}

func peerFailureReport(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, failureReportMaxBytes))
	if err != nil && len(raw) == 0 {
		return failureReportAbsent
	}

	report := strings.Join(strings.Fields(string(raw)), " ")
	if report == "" {
		return failureReportAbsent
	}

	return report
}
