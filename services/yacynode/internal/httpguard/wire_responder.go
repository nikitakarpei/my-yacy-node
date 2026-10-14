package httpguard

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const wireContentType = "text/plain; charset=UTF-8"

type RuntimeStatus interface {
	Version(ctx context.Context) string
	Uptime(ctx context.Context) int
}

type WireResponder struct {
	status RuntimeStatus
}

func NewWireResponder(status RuntimeStatus) WireResponder {
	return WireResponder{status: status}
}

func (r WireResponder) Write(
	ctx context.Context,
	w http.ResponseWriter,
	fields yacyproto.ResponseFields,
) {
	yacyproto.InjectResponseHeader(fields, r.status.Version(ctx), r.status.Uptime(ctx))
	writeResponseFields(ctx, w, fields)
}

func writeResponseFields(
	ctx context.Context,
	w http.ResponseWriter,
	fields yacyproto.ResponseFields,
) {
	w.Header().Set("Content-Type", wireContentType)
	if _, err := io.WriteString(w, fields.Encode()); err != nil {
		slog.WarnContext(ctx, "wire response write failed", slog.Any("error", err))
	}
}
