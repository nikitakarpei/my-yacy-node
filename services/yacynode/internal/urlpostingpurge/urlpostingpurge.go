// Package urlpostingpurge purges every posting that references a url when the
// url metadata is purged, inside the transaction that purges the metadata, so
// no posting outlives the url it describes.
package urlpostingpurge

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

type Observer interface {
	ObservePostingsPurgedWithURL(url yacymodel.URLHash, amountOfPostings int)
}

func New(
	references urlreferences.ReferenceQuery,
	purger rwipostings.PostingPurger,
	observer Observer,
) URLPostingPurge {
	return URLPostingPurge{references: references, purger: purger, observer: observer}
}
