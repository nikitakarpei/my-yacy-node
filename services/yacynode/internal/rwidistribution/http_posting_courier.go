package rwidistribution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const peerResponseMaxBodyBytes int64 = 1 << 20

var errPeerRequest = errors.New("peer request failed")

type postingOfferReceipt struct {
	Outcome    postingOfferOutcome
	RetryAfter time.Duration
}

type httpPostingCourier struct {
	client      *http.Client
	networkName string
	self        yacymodel.Hash
	urls        urlmeta.URLDirectory
}

func (c httpPostingCourier) Offer(ctx context.Context, offer postingOffer) postingOfferReceipt {
	endpoint, ok := offer.Peer.NetworkAddress()
	if !ok {
		slog.WarnContext(
			ctx,
			"posting offer not sent to peer without network address",
			slog.String("peer", offer.Peer.Hash.String()),
		)

		return postingOfferReceipt{Outcome: postingOfferUnaddressable}
	}

	resp, err := c.postTransferRWI(ctx, endpoint, offer)
	if err != nil {
		slog.WarnContext(
			ctx,
			"posting offer failed",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.Any("error", err),
		)

		return postingOfferReceipt{Outcome: postingOfferUnreachable}
	}

	switch resp.Result {
	case yacyproto.ResultOK:
		c.deliverUnknownURLs(ctx, endpoint, offer.Peer, resp.UnknownURL)

		return postingOfferReceipt{Outcome: postingOfferAccepted}
	case yacyproto.ResultBusy, yacyproto.ResultNotGranted:
		return postingOfferReceipt{Outcome: postingOfferDeferred, RetryAfter: resp.Pause}
	case yacyproto.ResultTooHighLoad:
		slog.WarnContext(
			ctx,
			"posting offer rejected by loaded peer",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.String("endpoint", endpoint),
		)

		return postingOfferReceipt{Outcome: postingOfferOverloaded}
	default:
		slog.WarnContext(
			ctx,
			"posting offer refused",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.String("result", string(resp.Result)),
		)

		return postingOfferReceipt{Outcome: postingOfferRefused}
	}
}

func (c httpPostingCourier) postTransferRWI(
	ctx context.Context,
	endpoint string,
	offer postingOffer,
) (yacyproto.TransferRWIResponse, error) {
	req := yacyproto.TransferRWIRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      offer.Peer.Hash,
		WordCount:   distinctWordCount(offer.Postings),
		EntryCount:  len(offer.Postings),
		Indexes:     offer.Postings,
		Key:         yacyproto.TransferRWIKey(len(offer.Postings)),
	}

	body, err := c.post(ctx, endpoint, yacyproto.PathTransferRWI, req.Form())
	if err != nil {
		return yacyproto.TransferRWIResponse{}, err
	}

	resp, err := yacyproto.ParseTransferRWIResponse(yacyproto.ParseMessage(body))
	if err != nil {
		return yacyproto.TransferRWIResponse{}, fmt.Errorf("%w: %w", errPeerRequest, err)
	}

	return resp, nil
}

func (c httpPostingCourier) deliverUnknownURLs(
	ctx context.Context,
	endpoint string,
	peer yacymodel.Seed,
	unknown []yacymodel.URLHash,
) {
	if len(unknown) == 0 {
		return
	}

	metadata, err := c.urls.MetadataByHash(ctx, unknown)
	if err != nil {
		slog.WarnContext(
			ctx,
			"unknown url metadata not read",
			slog.String("peer", peer.Hash.String()),
			slog.Any("error", err),
		)

		return
	}
	if len(metadata) == 0 {
		return
	}

	req := yacyproto.TransferURLRequest{
		NetworkName: c.networkName,
		Iam:         c.self,
		YouAre:      peer.Hash,
		URLCount:    len(metadata),
		URLs:        metadata,
	}

	if _, err := c.post(ctx, endpoint, yacyproto.PathTransferURL, req.Form()); err != nil {
		slog.WarnContext(
			ctx,
			"unknown url metadata not delivered",
			slog.String("peer", peer.Hash.String()),
			slog.Any("error", err),
		)
	}
}

func (c httpPostingCourier) post(
	ctx context.Context,
	endpoint, path string,
	form url.Values,
) (string, error) {
	target := url.URL{Scheme: "http", Host: endpoint, Path: path}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target.String(),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errPeerRequest, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errPeerRequest, err)
	}
	defer closeResponseBody(ctx, resp.Body, path)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", errPeerRequest, resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, peerResponseMaxBodyBytes))
	if err != nil {
		return "", fmt.Errorf("%w: %w", errPeerRequest, err)
	}

	return string(raw), nil
}

func distinctWordCount(postings []yacymodel.RWIPosting) int {
	words := make(map[yacymodel.Hash]struct{}, len(postings))
	for _, posting := range postings {
		words[posting.WordHash] = struct{}{}
	}

	return len(words)
}
