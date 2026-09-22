// Package rwiingress is the YaCy DHT ingress for reverse-word-index postings:
// it accepts a transferRWI request, checks that the request addresses this
// node and that this node accepts a remote index, and hands the batch to the
// posting receiver.
package rwiingress

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type transferRWIEndpoint struct {
	identity nodeidentity.Identity
	receiver rwiadmission.PostingReceiver
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

	receipt, err := e.receiver.Receive(ctx, req.Indexes)
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

func transferRWIResultFrom(receipt rwiadmission.Receipt) yacyproto.TransferRWIResult {
	if receipt.Busy {
		return yacyproto.ResultBusy
	}

	return yacyproto.ResultOK
}
