// Package searchquery holds the words a query asks for, its compound words, its
// exclusions and its language, the word hashes that address them on the DHT
// ring, and the spelling a held ranking answers to.
package searchquery

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Query struct {
	Words         []string
	CompoundWords []CompoundWord
	Exclusions    []string
	Language      string
}

func (q Query) String() string {
	spelled := make([]string, 0, len(q.Words)+len(q.Exclusions)+1)
	spelled = append(spelled, q.Words...)
	for _, exclusion := range q.Exclusions {
		spelled = append(spelled, "-"+exclusion)
	}
	if q.Language != "" {
		spelled = append(spelled, "lr:"+q.Language)
	}

	return strings.Join(spelled, " ")
}

func (q Query) WordHashes() []yacymodel.Hash {
	return hashesOf(q.Words)
}

func (q Query) HashesOfWordsAndCompoundWordsUpTo(compoundWordsCeiling int) []yacymodel.Hash {
	hashes := q.WordHashes()
	for _, compound := range q.CompoundWords[:min(max(compoundWordsCeiling, 0), len(q.CompoundWords))] {
		hashes = append(hashes, compound.Hash())
	}

	return hashes
}

func (q Query) ExclusionHashes() []yacymodel.Hash {
	return hashesOf(q.Exclusions)
}

func hashesOf(words []string) []yacymodel.Hash {
	hashes := make([]yacymodel.Hash, 0, len(words))
	for _, word := range words {
		hashes = append(hashes, yacymodel.WordHash(word))
	}

	return hashes
}
