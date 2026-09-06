// Package peeradmission answers inbound hello requests: it classifies the calling
// peer by probing it back, and returns the peers the roster saw most recently.
// On a confirmed back-ping it refreshes that caller's recency in the roster, but
// it never introduces a peer learned from an inbound request. A caller from
// another network is answered as a virgin peer, with no peers at all.
package peeradmission

import (
	"context"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type RuntimeStatus interface {
	NetworkName(ctx context.Context) string
	SelfSeed(ctx context.Context) yacymodel.Seed
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func MountHello(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	status RuntimeStatus,
	now func() time.Time,
	reachability reachableRoster,
	client *http.Client,
) {
	httpguard.Mount(
		router,
		yacyproto.PathHello,
		yacyproto.HelloEndpointMethods,
		yacyproto.ParseHelloRequest,
		helloEndpoint{
			identity:     identity,
			status:       status,
			now:          now,
			probe:        newCallerBackPing(client),
			reachability: reachability,
		}.Serve,
	)
}
