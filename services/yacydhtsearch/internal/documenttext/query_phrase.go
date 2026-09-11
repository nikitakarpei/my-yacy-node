package documenttext

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type queryPhrase struct {
	firstWord  yacymodel.Hash
	secondWord yacymodel.Hash
}

func queryPhrasesOf(queryWords []yacymodel.Hash) map[queryPhrase]struct{} {
	queryPhrases := map[queryPhrase]struct{}{}
	for place := range len(queryWords) - 1 {
		queryPhrases[queryPhrase{
			firstWord:  queryWords[place],
			secondWord: queryWords[place+1],
		}] = struct{}{}
	}

	return queryPhrases
}
