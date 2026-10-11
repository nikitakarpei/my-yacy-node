// Package stopwordhashes holds the word hashes of the stopwords of each language
// the stopwords library lists, so a posting, which names its word only by hash,
// can be told to be a stopword posting.
package stopwordhashes

import (
	"github.com/nikitakarpei/yacy-rwi-node/stopwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type StopwordHashes struct {
	hashesPerLanguage map[string]map[yacymodel.Hash]struct{}
}

func New() StopwordHashes {
	hashesPerLanguage := make(map[string]map[yacymodel.Hash]struct{})
	for _, language := range stopwords.Languages() {
		hashesPerLanguage[language] = hashesOfTheStopwordsOf(language)
	}

	return StopwordHashes{hashesPerLanguage: hashesPerLanguage}
}

func hashesOfTheStopwordsOf(language string) map[yacymodel.Hash]struct{} {
	stopwordsOfTheLanguage, _ := stopwords.Of(language)
	hashes := make(map[yacymodel.Hash]struct{})
	for _, word := range stopwordsOfTheLanguage.Words() {
		hashes[yacymodel.WordHash(word)] = struct{}{}
	}

	return hashes
}

func (s StopwordHashes) Contains(word yacymodel.Hash, language yacymodel.Language) bool {
	_, contained := s.hashesPerLanguage[language.String()][word]

	return contained
}
