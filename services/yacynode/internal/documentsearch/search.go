// Package documentsearch finds documents containing query terms, orders them by
// relevance, and reports how many documents matched each term.
package documentsearch

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func MountSearch(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	index rwipostings.PostingIndex,
	documents urlmeta.URLDirectory,
	maxPostingsPerTerm int,
) {
	endpoint := searchEndpoint{
		identity: identity,
		searcher: searcher{
			index:              index,
			documentDirectory:  documents,
			maxPostingsPerTerm: maxPostingsPerTerm,
		},
	}

	httpguard.Mount(
		router,
		yacyproto.PathSearch,
		yacyproto.SearchEndpointMethods,
		yacyproto.ParseSearchRequest,
		endpoint.Serve,
	)
}
