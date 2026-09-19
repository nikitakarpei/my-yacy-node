package queryanswers

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type LinkCounts struct {
	LocalLinks    int
	ExternalLinks int
}

func LinkCountsFrom(posting yacymodel.RWIPosting) LinkCounts {
	return LinkCounts{
		LocalLinks:    posting.LocalLinks,
		ExternalLinks: posting.ExternalLinks,
	}
}
