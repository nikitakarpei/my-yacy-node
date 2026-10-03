// Package wordroles splits the words of a query around its lead. Listing words
// are asked whole and their answers list the documents to match; matching
// words are asked only among those documents.
package wordroles

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Roles struct {
	ListingWords  []yacymodel.Hash
	MatchingWords []yacymodel.Hash
}

func From(lead leadingword.Lead, query searchquery.Query) Roles {
	leadingWord, led := lead.Word.Get()
	if !led {
		return Roles{ListingWords: wordsAndCompoundWordsOf(query)}
	}

	return rolesAround(leadingWord, query)
}

func wordsAndCompoundWordsOf(query searchquery.Query) []yacymodel.Hash {
	words := slices.Clone(query.WordHashes())
	for _, compoundWord := range query.CompoundWords {
		words = append(words, compoundWord.Hash())
	}

	return words
}

func rolesAround(leadingWord yacymodel.Hash, query searchquery.Query) Roles {
	roles := Roles{ListingWords: []yacymodel.Hash{leadingWord}}
	for _, queryWord := range query.WordHashes() {
		if queryWord == leadingWord {
			continue
		}
		roles.MatchingWords = append(roles.MatchingWords, queryWord)
	}
	for _, compoundWord := range query.CompoundWords {
		if slices.Contains(compoundWord.PartHashes(), leadingWord) {
			roles.ListingWords = append(roles.ListingWords, compoundWord.Hash())

			continue
		}
		roles.MatchingWords = append(roles.MatchingWords, compoundWord.Hash())
	}

	return roles
}
