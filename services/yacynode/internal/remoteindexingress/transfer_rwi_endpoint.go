// Package remoteindexingress is the YaCy DHT ingress for this node's remote
// index: it accepts transferRWI and transferURL requests, checks that each
// addresses this node and that this node accepts a remote index, and hands the
// postings or the URL metadata to remote index admission with the sender's
// network.
package remoteindexingress

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type transferRWIEndpoint struct {
	identity nodeidentity.Identity
	receiver remoteindexadmission.PostingReceiver
}

func (e transferRWIEndpoint) Serve(
	ctx context.Context,
	req yacyproto.TransferRWIRequest,
) (yacyproto.TransferRWIResponse, error) {
	resp := yacyproto.TransferRWIResponse{}

	if !e.identity.Addresses(req.NetworkName, req.YouAre) {
		resp.Result = yacyproto.ResultWrongTarget

		return resp, nil
	}

	if !e.identity.Capabilities.AcceptRemoteIndex {
		resp.Result = yacyproto.ResultNotGranted
		slog.DebugContext(ctx, "transfer rwi refused: remote index not accepted",
			slog.Int("entryCount", len(req.Indexes)),
		)

		return resp, nil
	}

	slog.DebugContext(ctx, "transfer rwi request received",
		slog.Int("wordCount", req.WordCount),
		slog.Int("entryCount", req.EntryCount),
		slog.Int("receivedEntryCount", len(req.Indexes)),
	)

	origin := rwiorigin.Of(httpguard.RemoteAddr(ctx))
	receipt, err := e.receiver.Receive(ctx, origin, pagePostingsFrom(ctx, req.Indexes))
	if err != nil {
		return yacyproto.TransferRWIResponse{}, fmt.Errorf("receive rwi: %w", err)
	}

	resp.Result = transferRWIResultFrom(receipt)
	resp.Pause = receipt.Pause
	resp.UnknownURL = receipt.UnknownURL

	slog.DebugContext(ctx, "transfer rwi answered",
		slog.String("result", string(resp.Result)),
		slog.Int("unknownUrlCount", len(receipt.UnknownURL)),
	)

	return resp, nil
}

func pagePostingsFrom(
	ctx context.Context,
	postings []yacymodel.RWIPosting,
) []yacymodel.PagePostings {
	pagePostingsPerURL, repeatedPostings := yacymodel.PagePostingsPerURLFrom(postings)
	for _, posting := range repeatedPostings {
		reportRepeatedPosting(ctx, posting)
	}

	return pagePostingsPerURL
}

func reportRepeatedPosting(ctx context.Context, posting yacymodel.RWIPosting) {
	slog.WarnContext(ctx, "transfer rwi repeated posting dropped",
		slog.String("url", posting.URLHash.String()),
		slog.String("word", posting.WordHash.String()),
	)
}

func transferRWIResultFrom(
	receipt remoteindexadmission.PostingReceipt,
) yacyproto.TransferRWIResult {
	if receipt.Busy {
		return yacyproto.ResultBusy
	}

	return yacyproto.ResultOK
}
