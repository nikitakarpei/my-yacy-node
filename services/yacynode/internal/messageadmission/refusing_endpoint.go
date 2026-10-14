package messageadmission

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type refusingEndpoint struct{}

func (refusingEndpoint) Serve(
	ctx context.Context,
	_ yacyproto.MessageRequest,
) (yacyproto.MessageResponse, error) {
	slog.DebugContext(ctx, "message refused, the node has no inbox")

	return yacyproto.MessageResponse{Response: yacyproto.MessageResponseRefused}, nil
}
