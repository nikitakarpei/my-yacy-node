// Package nodestatus owns the node's runtime status: its self-seed, the
// version/uptime header every endpoint echoes, and the query.html capacity
// answers. Its published port, RuntimeStatus, is the only surface other modules
// import. Live counts arrive through the RWICounter and URLCounter ports, which
// read in the transaction nodestatus opens, so nodestatus never reads another
// module's schema.
package nodestatus

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type RuntimeStatus interface {
	NetworkName(ctx context.Context) string
	Version(ctx context.Context) string
	Uptime(ctx context.Context) int
	SelfSeed(ctx context.Context) yacymodel.Seed
}

type RWICounter interface {
	RWICount(tx *vault.Txn) (int, error)
}

type ReferencedURLCounter interface {
	ReferencedURLCount(tx *vault.Txn) (int, error)
}

type URLCounter interface {
	Count(tx *vault.Txn) (int, error)
}

func NewRuntimeStatus(
	id nodeidentity.Identity,
	now func() time.Time,
	v *vault.Vault,
	rwi RWICounter,
	urls URLCounter,
) RuntimeStatus {
	return runtimeStatus{
		id:    id,
		base:  baseSeed(id),
		now:   now,
		vault: v,
		rwi:   rwi,
		urls:  urls,
	}
}

//nolint:revive // argument-limit
func MountQuery(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	v *vault.Vault,
	rwi RWICounter,
	references ReferencedURLCounter,
	urls URLCounter,
) {
	httpguard.Mount(
		router,
		yacyproto.PathQuery,
		yacyproto.QueryEndpointMethods,
		yacyproto.ParseQueryRequest,
		queryEndpoint{
			identity:   identity,
			vault:      v,
			rwi:        rwi,
			references: references,
			urls:       urls,
		}.Serve,
	)
}
