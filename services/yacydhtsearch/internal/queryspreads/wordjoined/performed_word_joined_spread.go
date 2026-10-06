package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentsperword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedWordJoinedSpread struct {
	DocumentAsks                        documentasks.Performed
	AmountOfQueryWords                  int
	AmountOfCompoundWords               int
	AmountOfQueryWordsHeldByNoPeer      int
	AmountOfDocumentsOfTheLeadingWord   int
	AmountOfJoinedDocuments             int
	AmountOfJoinedDocumentsWithMetadata int
	URLMetadataAsks                     urlmetadataasks.Performed
	TimeSpent                           time.Duration
}

//nolint:revive // argument-limit: the report takes each part the spread performed
func performedWordJoinedSpreadFrom(
	answered []documentasks.AnsweredWordPartition,
	holders documentholders.Holders,
	documentsPerWord documentsperword.DocumentsPerWord,
	lead yacymodel.Optional[leadingword.Lead],
	joinedDocuments yacymodel.URLHashes,
	urlMetadataAnswers urlmetadataasks.Answers,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	return PerformedWordJoinedSpread{
		DocumentAsks:                      documentasks.PerformedFrom(answered),
		AmountOfQueryWords:                documentsPerWord.AmountOfQueryWords(),
		AmountOfCompoundWords:             documentsPerWord.AmountOfCompoundWords(),
		AmountOfQueryWordsHeldByNoPeer:    documentsPerWord.AmountOfQueryWordsHeldByNoPeer(),
		AmountOfDocumentsOfTheLeadingWord: len(documentsPerWord.OfTheLeadingWord(wordOf(lead))),
		AmountOfJoinedDocuments:           len(joinedDocuments),
		AmountOfJoinedDocumentsWithMetadata: len(joinedDocuments) -
			len(holders.WithoutMetadataAmong(joinedDocuments)),
		URLMetadataAsks: urlmetadataasks.PerformedFrom(urlMetadataAnswers),
		TimeSpent:       timeSpent,
	}
}

func wordOf(lead yacymodel.Optional[leadingword.Lead]) yacymodel.Optional[yacymodel.Hash] {
	chosenLead, chosen := lead.Get()
	if !chosen {
		return yacymodel.None[yacymodel.Hash]()
	}

	return yacymodel.Some(chosenLead.Word)
}
