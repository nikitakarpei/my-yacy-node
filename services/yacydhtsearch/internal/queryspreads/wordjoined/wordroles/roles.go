// Package wordroles splits the words of a query around its lead: the lead and
// its compound words, which list the documents to match, and the other words,
// which are matched against those documents.
package wordroles

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Roles struct {
	LeadAndItsCompoundWords []yacymodel.Hash
	OtherWords              []yacymodel.Hash
}

func Around(lead leadingword.Lead, query searchquery.Query) Roles {
	roles := Roles{LeadAndItsCompoundWords: []yacymodel.Hash{lead.Word}}
	for _, queryWord := range query.WordHashes() {
		if queryWord == lead.Word {
			continue
		}
		roles.OtherWords = append(roles.OtherWords, queryWord)
	}
	for _, compoundWord := range query.CompoundWords {
		if slices.Contains(compoundWord.PartHashes(), lead.Word) {
			roles.LeadAndItsCompoundWords = append(
				roles.LeadAndItsCompoundWords,
				compoundWord.Hash(),
			)

			continue
		}
		roles.OtherWords = append(roles.OtherWords, compoundWord.Hash())
	}

	return roles
}
