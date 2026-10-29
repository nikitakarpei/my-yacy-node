// Package peeradmission answers inbound hello requests: it classifies the calling
// peer by probing it back, and returns the peers the roster saw most recently.
// On a confirmed back-ping it gives the caller's seed to the roster as a
// second-hand seed, because the caller can send a copy of another peer's seed.
// A caller from another network is answered as a virgin peer, with no peers at
// all.
package peeradmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type RuntimeStatus interface {
	NetworkName(ctx context.Context) string
	SelfSeed(ctx context.Context) yacymodel.Seed
}

type Clock interface {
	Now() time.Time
	After(timeout time.Duration, expire func()) (stop func())
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func MountHello(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	status RuntimeStatus,
	clock Clock,
	reachability reachableRoster,
	walker peerwire.Walker,
) {
	httpguard.MountFields(
		router,
		yacyproto.PathHello,
		yacyproto.HelloEndpointMethods,
		yacyproto.ParseHelloRequest,
		helloEndpoint{
			identity:     identity,
			status:       status,
			clock:        clock,
			probe:        newCallerBackPing(walker, reachability, clock),
			reachability: reachability,
		}.Serve,
	)
}
