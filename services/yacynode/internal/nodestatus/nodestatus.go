// Package nodestatus owns the node's runtime status: its self-seed, the
// version/uptime header every endpoint echoes, and the query.html capacity
// answers. Its published port, RuntimeStatus, is the only surface other modules
// import. Live counts arrive through the WordAmountEstimate, PostingAmountCounter
// and URLCounter ports, which read in the transaction nodestatus opens, so
// nodestatus never reads another module's schema.
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

type WordAmountEstimate interface {
	EstimatedAmountOfWords(tx *vault.Txn) (int, error)
}

type PostingAmountCounter interface {
	AmountOfPostingsOf(tx *vault.Txn, word yacymodel.Hash) (int, error)
}

type URLCounter interface {
	Count(tx *vault.Txn) (int, error)
}

func NewRuntimeStatus(
	id nodeidentity.Identity,
	now func() time.Time,
	v *vault.Vault,
	words WordAmountEstimate,
	urls URLCounter,
) RuntimeStatus {
	return runtimeStatus{
		id:    id,
		base:  baseSeed(id),
		now:   now,
		vault: v,
		words: words,
		urls:  urls,
	}
}

//nolint:revive // argument-limit
func MountQuery(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	now func() time.Time,
	v *vault.Vault,
	words WordAmountEstimate,
	postingAmounts PostingAmountCounter,
	urls URLCounter,
) {
	httpguard.MountFields(
		router,
		yacyproto.PathQuery,
		yacyproto.QueryEndpointMethods,
		yacyproto.ParseQueryRequest,
		queryEndpoint{
			identity:       identity,
			now:            now,
			vault:          v,
			words:          words,
			postingAmounts: postingAmounts,
			urls:           urls,
		}.Serve,
	)
}
