package pagepublication

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

func acceptsSourceFormat(
	derivation crawlcapability.ContentDerivation,
	format crawlcapability.PageContentFormat,
) bool {
	return slices.Contains(derivation.SourceFormats(), format)
}
