package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentsperword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedWordJoinedSpread struct {
	AmountOfQueryWords                  int
	AmountOfCompoundWords               int
	AmountOfQueryWordsHeldByNoPeer      int
	AmountOfDocumentsOfTheLeadingWord   int
	AmountOfJoinedDocuments             int
	AmountOfJoinedDocumentsWithMetadata int
	TimeSpent                           time.Duration
}

func performedWordJoinedSpreadFrom(
	holders documentholders.Holders,
	documentsPerWord documentsperword.DocumentsPerWord,
	lead yacymodel.Optional[leadingword.Lead],
	joinedDocuments yacymodel.URLHashes,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	return PerformedWordJoinedSpread{
		AmountOfQueryWords:                documentsPerWord.AmountOfQueryWords(),
		AmountOfCompoundWords:             documentsPerWord.AmountOfCompoundWords(),
		AmountOfQueryWordsHeldByNoPeer:    documentsPerWord.AmountOfQueryWordsHeldByNoPeer(),
		AmountOfDocumentsOfTheLeadingWord: len(documentsPerWord.OfTheLeadingWord(wordOf(lead))),
		AmountOfJoinedDocuments:           len(joinedDocuments),
		AmountOfJoinedDocumentsWithMetadata: len(joinedDocuments) -
			len(holders.WithoutMetadataAmong(joinedDocuments)),
		TimeSpent: timeSpent,
	}
}

func wordOf(lead yacymodel.Optional[leadingword.Lead]) yacymodel.Optional[yacymodel.Hash] {
	chosenLead, chosen := lead.Get()
	if !chosen {
		return yacymodel.None[yacymodel.Hash]()
	}

	return yacymodel.Some(chosenLead.Word)
}
