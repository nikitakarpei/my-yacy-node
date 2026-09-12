package httpguard

import (
	"context"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type WireRouter struct {
	mux  *http.ServeMux
	gate WireGate
}

func NewWireRouter(mux *http.ServeMux, gate WireGate) WireRouter {
	return WireRouter{mux: mux, gate: gate}
}

func MountMessage[Req any, Resp MessageResponse](
	router WireRouter,
	path string,
	methods yacyproto.EndpointMethodSet,
	parse func(ctx context.Context, form url.Values) (Req, error),
	serve func(ctx context.Context, req Req) (Resp, error),
) {
	router.mux.Handle(path, ServeMessage(router.gate, methods, parse, serve))
}

func MountFeed[Req any, Resp FeedResponse](
	router WireRouter,
	path string,
	methods yacyproto.EndpointMethodSet,
	parse func(ctx context.Context, form url.Values) (Req, error),
	serve func(ctx context.Context, req Req) (Resp, error),
) {
	router.mux.Handle(path, ServeFeed(router.gate, methods, parse, serve))
}
