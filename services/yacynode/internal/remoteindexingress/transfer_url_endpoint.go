package remoteindexingress

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type transferURLEndpoint struct {
	identity nodeidentity.Identity
	receiver remoteindexadmission.URLMetadataReceiver
}

func (e transferURLEndpoint) Serve(
	ctx context.Context,
	req yacyproto.TransferURLRequest,
) (yacyproto.TransferURLResponse, error) {
	resp := yacyproto.TransferURLResponse{}

	if !e.identity.Addresses(req.NetworkName, req.YouAre) {
		resp.Result = yacyproto.ResultWrongTarget

		return resp, nil
	}

	if !e.identity.Capabilities.AcceptRemoteIndex {
		resp.Result = yacyproto.ResultErrorNotGranted
		slog.DebugContext(ctx, "transfer url refused: remote index not accepted",
			slog.Int("urlCount", len(req.URLs)),
		)

		return resp, nil
	}

	receipt, err := e.receiver.Receive(ctx, rwiorigin.Of(httpguard.RemoteAddr(ctx)), req.URLs)
	if err != nil {
		return yacyproto.TransferURLResponse{}, fmt.Errorf("receive url: %w", err)
	}

	if receipt.Busy {
		resp.Result = yacyproto.ResultErrorNotGranted
	} else {
		resp.Result = yacyproto.ResultOK
	}
	resp.Double = receipt.KnownURLs

	slog.DebugContext(ctx, "transfer url stored",
		slog.Bool("busy", receipt.Busy),
		slog.Int("knownUrls", receipt.KnownURLs),
	)

	return resp, nil
}
