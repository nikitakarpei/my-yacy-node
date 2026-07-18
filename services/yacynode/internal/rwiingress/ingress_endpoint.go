// Package rwiingress is the YaCy DHT ingress for reverse-word-index postings:
// it accepts a transferRWI request, projects each property-form posting onto the
// domain type, and hands the batch to the posting receiver. It is the only place
// the node reads YaCy's posting wire vocabulary on the way in.
package rwiingress

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const msgPostingDiscarded = "transfer rwi posting discarded"

type transferRWIEndpoint struct {
	identity nodeidentity.Identity
	intake   rwipostings.PostingReceiver
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

	postings := projectPostings(ctx, req.Indexes)

	slog.DebugContext(ctx, "transfer rwi request accepted",
		slog.Int("wordCount", req.WordCount),
		slog.Int("entryCount", req.EntryCount),
		slog.Int("acceptedEntryCount", len(postings)),
	)

	receipt, err := e.intake.Receive(ctx, postings)
	if err != nil {
		return yacyproto.TransferRWIResponse{}, fmt.Errorf("receive rwi: %w", err)
	}

	if receipt.Busy {
		resp.Result = yacyproto.ResultBusy
	} else {
		resp.Result = yacyproto.ResultOK
	}
	resp.Pause = receipt.Pause
	resp.UnknownURL = receipt.UnknownURL

	slog.DebugContext(ctx, "transfer rwi stored",
		slog.Bool("busy", receipt.Busy),
		slog.Int("unknownUrlCount", len(receipt.UnknownURL)),
	)

	return resp, nil
}

func projectPostings(
	ctx context.Context,
	wireForms []yacymodel.RWIPostingWireForm,
) []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(wireForms))
	for _, wireForm := range wireForms {
		posting, err := wireForm.Domain()
		if err != nil {
			slog.WarnContext(ctx, msgPostingDiscarded, slog.Any("error", err))
			continue
		}
		postings = append(postings, posting)
	}

	return postings
}
