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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const offerMaxBodyBytes int64 = 1 << 20

var errOfferFailed = errors.New("posting offer failed")

type offerOutcome struct {
	Accepted   bool
	RetryAfter time.Duration
}

type httpPostingCourier struct {
	client      *http.Client
	networkName string
	self        yacymodel.Hash
	roster      peerroster.Roster
	ledger      *replicaLedger
	urls        urlmeta.URLDirectory
	observer    OfferObserver
}

func (c httpPostingCourier) Offer(ctx context.Context, offer postingOffer) offerOutcome {
	endpoint, ok := offer.Peer.NetworkAddress()
	if !ok {
		c.roster.ConfirmUnreachable(ctx, offer.Peer.Hash)
		c.observer.ObserveOffer(offerResultUnreachable, len(offer.Postings))

		return offerOutcome{}
	}

	resp, err := c.postTransferRWI(ctx, endpoint, offer)
	if err != nil {
		c.roster.ConfirmUnreachable(ctx, offer.Peer.Hash)
		c.observer.ObserveOffer(offerResultError, len(offer.Postings))
		slog.WarnContext(
			ctx,
			"posting offer failed",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.Any("error", err),
		)

		return offerOutcome{}
	}

	c.observer.ObserveOffer(string(resp.Result), len(offer.Postings))

	switch resp.Result {
	case yacyproto.ResultOK:
		c.recordAccepted(ctx, offer)
		c.deliverUnknownURLs(ctx, endpoint, offer.Peer, resp.UnknownURL)

		return offerOutcome{Accepted: true}
	case yacyproto.ResultBusy, yacyproto.ResultNotGranted:
		return offerOutcome{RetryAfter: resp.Pause}
	case yacyproto.ResultTooHighLoad:
		slog.WarnContext(
			ctx,
			"posting offer rejected by loaded peer",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.String("endpoint", endpoint),
		)

		return offerOutcome{}
	default:
		slog.WarnContext(
			ctx,
			"posting offer refused",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.String("endpoint", endpoint),
			slog.String("result", string(resp.Result)),
		)
		c.roster.ConfirmUnreachable(ctx, offer.Peer.Hash)

		return offerOutcome{}
	}
}

func (c httpPostingCourier) recordAccepted(ctx context.Context, offer postingOffer) {
	for _, posting := range offer.Postings {
		word, targetURL := posting.WordHash, posting.URLHash.Hash()
		if err := c.ledger.RecordAccepted(ctx, word, targetURL, offer.Peer.Hash); err != nil {
			slog.WarnContext(
				ctx,
				"replica not recorded",
				slog.String("peer", offer.Peer.Hash.String()),
				slog.String("word", word.String()),
				slog.String("url", targetURL.String()),
				slog.Any("error", err),
			)
		}
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
		Key:         offerKey(offer.Postings),
	}

	body, err := c.post(ctx, endpoint, yacyproto.PathTransferRWI, req.Form())
	if err != nil {
		return yacyproto.TransferRWIResponse{}, err
	}

	resp, err := yacyproto.ParseTransferRWIResponse(yacyproto.ParseMessage(body))
	if err != nil {
		return yacyproto.TransferRWIResponse{}, fmt.Errorf("%w: %w", errOfferFailed, err)
	}

	return resp, nil
}

func (c httpPostingCourier) deliverUnknownURLs(
	ctx context.Context,
	endpoint string,
	peer yacymodel.Seed,
	unknown []yacymodel.Hash,
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
		return "", fmt.Errorf("%w: %w", errOfferFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errOfferFailed, err)
	}
	defer closeResponseBody(ctx, resp.Body, path)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", errOfferFailed, resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, offerMaxBodyBytes))
	if err != nil {
		return "", fmt.Errorf("%w: %w", errOfferFailed, err)
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

func offerKey(postings []yacymodel.RWIPosting) string {
	return yacyproto.MagicMD5("", "", fmt.Sprintf("%d", len(postings)))
}
