// Package documentsearch mounts the endpoint that finds documents containing
// query terms, orders them by relevance, and reports how many documents matched
// each term.
package documentsearch

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchendpoint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

func MountSearch(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	index rwipostings.PostingIndex,
	documents urlmeta.URLDirectory,
	maxPostingsPerTerm int,
) {
	searchendpoint.Mount(
		router,
		identity,
		searchresult.New(index, documents, maxPostingsPerTerm),
	)
}
