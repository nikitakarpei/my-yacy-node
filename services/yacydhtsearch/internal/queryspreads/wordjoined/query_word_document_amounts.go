package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func rarestQueryWordAmong(
	words []yacymodel.Hash,
	documentAmounts map[yacymodel.Hash]int,
) yacymodel.Optional[yacymodel.Hash] {
	rarestQueryWord := yacymodel.None[yacymodel.Hash]()
	fewestDocuments := 0
	for _, word := range words {
		amountOfDocuments, remembered := documentAmounts[word]
		if !remembered {
			return yacymodel.None[yacymodel.Hash]()
		}
		if rarestQueryWord.Present() && amountOfDocuments >= fewestDocuments {
			continue
		}
		rarestQueryWord = yacymodel.Some(word)
		fewestDocuments = amountOfDocuments
	}

	return rarestQueryWord
}
