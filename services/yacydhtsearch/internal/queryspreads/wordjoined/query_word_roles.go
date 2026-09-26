package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryWordRoles struct {
	wordsOfTheDocumentsToMatch []yacymodel.Hash
	otherWords                 []yacymodel.Hash
}

func queryWordRolesAround(
	leadingQueryWord yacymodel.Hash,
	query searchquery.Query,
) queryWordRoles {
	roles := queryWordRoles{
		wordsOfTheDocumentsToMatch: []yacymodel.Hash{leadingQueryWord},
	}
	for _, queryWord := range query.WordHashes() {
		if queryWord == leadingQueryWord {
			continue
		}
		roles.otherWords = append(roles.otherWords, queryWord)
	}
	for _, compoundWord := range query.CompoundWords {
		if slices.Contains(compoundWord.WordHashes(), leadingQueryWord) {
			roles.wordsOfTheDocumentsToMatch = append(
				roles.wordsOfTheDocumentsToMatch, compoundWord.Hash(),
			)

			continue
		}
		roles.otherWords = append(roles.otherWords, compoundWord.Hash())
	}

	return roles
}
