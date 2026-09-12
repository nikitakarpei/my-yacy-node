package httpguard

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const feedContentType = "text/xml; charset=UTF-8"

type MessageResponse interface {
	Encode() yacyproto.Message
}

type FeedResponse interface {
	Encode() []byte
}

type WireGate struct {
	Guard   RequestGuard
	Respond WireResponder
	Address ClientAddressResolver
}

func ServeMessage[Req any, Resp MessageResponse](
	gate WireGate,
	methods yacyproto.EndpointMethodSet,
	parse func(ctx context.Context, form url.Values) (Req, error),
	serve func(ctx context.Context, req Req) (Resp, error),
) http.Handler {
	return serveWritten(gate, methods, parse, serve,
		func(ctx context.Context, w http.ResponseWriter, resp Resp) {
			gate.Respond.Write(ctx, w, resp.Encode())
		},
	)
}

func serveWritten[Req any, Resp any](
	gate WireGate,
	methods yacyproto.EndpointMethodSet,
	parse func(ctx context.Context, form url.Values) (Req, error),
	serve func(ctx context.Context, req Req) (Resp, error),
	write func(ctx context.Context, w http.ResponseWriter, resp Resp),
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		form, ctx, cancel, ok := gate.Guard.Parse(w, r, methods)
		if !ok {
			return
		}
		defer cancel()

		ctx = WithRemoteAddr(ctx, gate.Address.Resolve(r))

		req, err := parse(ctx, form)
		if err != nil {
			FailBadRequest(ctx, w, err)

			return
		}

		resp, err := serve(ctx, req)
		if err != nil {
			failInternal(ctx, w, r.URL.Path, err)

			return
		}

		write(ctx, w, resp)
	})
}

func failInternal(ctx context.Context, w http.ResponseWriter, path string, err error) {
	slog.ErrorContext(ctx, "request failed",
		slog.String("path", path),
		slog.Any("error", err),
	)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func ServeFeed[Req any, Resp FeedResponse](
	gate WireGate,
	methods yacyproto.EndpointMethodSet,
	parse func(ctx context.Context, form url.Values) (Req, error),
	serve func(ctx context.Context, req Req) (Resp, error),
) http.Handler {
	return serveWritten(gate, methods, parse, serve, writeFeed[Resp])
}

func writeFeed[Resp FeedResponse](ctx context.Context, w http.ResponseWriter, resp Resp) {
	w.Header().Set("Content-Type", feedContentType)
	if _, err := w.Write(resp.Encode()); err != nil {
		slog.WarnContext(ctx, "feed response write failed", slog.Any("error", err))
	}
}
