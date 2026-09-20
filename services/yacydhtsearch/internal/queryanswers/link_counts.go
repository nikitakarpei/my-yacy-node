package queryanswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func LinkCountsFrom(posting yacymodel.RWIPosting) pagecontents.LinkCounts {
	return pagecontents.LinkCounts{
		LocalLinks:    posting.LocalLinks,
		ExternalLinks: posting.ExternalLinks,
	}
}
