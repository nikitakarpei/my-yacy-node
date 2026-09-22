package queryanswers

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type CompoundQueryWord struct {
	Word       yacymodel.Hash
	FirstWord  yacymodel.Hash
	SecondWord yacymodel.Hash
}

func CompoundQueryWordsFrom(terms []string) []CompoundQueryWord {
	if len(terms) < 2 {
		return nil
	}
	compoundQueryWords := make([]CompoundQueryWord, 0, len(terms)-1)
	for place := range len(terms) - 1 {
		compoundQueryWords = append(compoundQueryWords, CompoundQueryWord{
			Word:       yacymodel.WordHash(terms[place] + terms[place+1]),
			FirstWord:  yacymodel.WordHash(terms[place]),
			SecondWord: yacymodel.WordHash(terms[place+1]),
		})
	}

	return compoundQueryWords
}
