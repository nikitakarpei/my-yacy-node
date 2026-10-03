package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/matchingwords"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordholdings"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedWordJoinedSpread struct {
	WordAsks                            wordasks.Performed
	AmountOfQueryWords                  int
	AmountOfCompoundWords               int
	AmountOfQueryWordsHeldByNoPeer      int
	AmountOfDocumentsOfTheLeadingWord   int
	MatchingWordAskKindPerPartition     matchingwords.KindPerPartition
	AmountOfJoinedDocuments             int
	AmountOfJoinedDocumentsWithMetadata int
	URLMetadataAsks                     urlmetadataasks.Performed
	TimeSpent                           time.Duration
}

//nolint:revive // argument-limit: the report takes each part the spread performed
func performedWordJoinedSpreadFrom(
	wordAnswers wordasks.Answers,
	holdings wordholdings.Holdings,
	lead leadingword.Lead,
	matchingWordAskKindPerPartition matchingwords.KindPerPartition,
	joinedDocuments yacymodel.URLHashes,
	documentsWithoutMetadata yacymodel.URLHashes,
	urlMetadata urlmetadataasks.Answers,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	return PerformedWordJoinedSpread{
		WordAsks:                       wordasks.PerformedFrom(wordAnswers),
		AmountOfQueryWords:             holdings.AmountOfQueryWords(),
		AmountOfCompoundWords:          holdings.AmountOfCompoundWords(),
		AmountOfQueryWordsHeldByNoPeer: holdings.AmountOfQueryWordsHeldByNoPeer(),
		AmountOfDocumentsOfTheLeadingWord: len(
			holdings.DocumentsOfTheLeadingWord(lead.Word),
		),
		MatchingWordAskKindPerPartition:     matchingWordAskKindPerPartition,
		AmountOfJoinedDocuments:             len(joinedDocuments),
		AmountOfJoinedDocumentsWithMetadata: len(joinedDocuments) - len(documentsWithoutMetadata),
		URLMetadataAsks:                     urlmetadataasks.PerformedFrom(urlMetadata),
		TimeSpent:                           timeSpent,
	}
}
