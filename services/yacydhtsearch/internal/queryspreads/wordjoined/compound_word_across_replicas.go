package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"

type compoundWordAcrossReplicas struct {
	searchquery.CompoundWord
	wordAcrossReplicas
}
