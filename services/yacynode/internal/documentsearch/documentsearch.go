// Package documentsearch mounts the endpoint that finds documents containing
// query terms, orders them by relevance, and reports how many postings this
// node holds for each term.
package documentsearch

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchendpoint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingamount"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

//nolint:revive // argument-limit: the mounted search names its collaborators explicitly.
func MountSearch(
	v *vault.Vault,
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	postingIndex rwipostings.PostingIndex,
	impactOrder rwipostingimpactorder.ImpactOrderQuery,
	postingAmounts rwipostingamount.PostingAmountQuery,
	documentDirectory searchresult.DocumentDirectory,
	metrics *searchmetrics.SearchMetrics,
	partitions yacymodel.DHTRingPartitions,
	mostRelevantDocumentsPerTerm int,
) {
	searchendpoint.Mount(
		router,
		identity,
		searchresult.New(
			v,
			documentmatch.New(postingIndex, impactOrder),
			termmatch.New(postingIndex, impactOrder, mostRelevantDocumentsPerTerm),
			postingAmounts,
			documentDirectory,
		),
		metrics,
		partitions,
	)
}
