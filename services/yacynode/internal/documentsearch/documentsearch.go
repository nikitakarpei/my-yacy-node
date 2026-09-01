// Package documentsearch mounts the endpoint that finds documents containing
// query terms, orders them by relevance, and reports how many documents matched
// each term.
package documentsearch

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchendpoint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

//nolint:revive // argument-limit: the mounted search names its collaborators explicitly.
func MountSearch(
	v *vault.Vault,
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	index rwipostings.PostingIndex,
	documents searchresult.DocumentDirectory,
	maxPostingsPerTerm int,
	metrics *searchmetrics.SearchMetrics,
	partitions yacymodel.DHTRingPartitions,
) {
	searchendpoint.Mount(
		router,
		identity,
		searchresult.New(v, index, documents, maxPostingsPerTerm),
		metrics,
		partitions,
	)
}
