package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedWordJoinedSpread struct {
	AmountOfQueryWords                int
	AmountOfQueryWordsHeldByNoPeer    int
	AmountOfDocumentsOfTheLeadingWord yacymodel.Optional[int]
	AmountOfJoinedDocuments           int
	TimeSpent                         time.Duration
}

//nolint:revive // argument-limit: the report reads each unit of the spread it describes
func performedWordJoinedSpreadFrom(
	query searchquery.Query,
	measurement *documentamounts.Measurement,
	lead yacymodel.Optional[leadingword.Lead],
	joinedDocuments yacymodel.URLHashes,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	performed := PerformedWordJoinedSpread{
		AmountOfQueryWords:             len(query.WordHashes()),
		AmountOfQueryWordsHeldByNoPeer: measurement.AmountOfQueryWordsHeldByNoPeer(),
		AmountOfJoinedDocuments:        joinedDocuments.Amount(),
		TimeSpent:                      timeSpent,
	}
	if chosenLead, led := lead.Get(); led {
		performed.AmountOfDocumentsOfTheLeadingWord = yacymodel.Some(
			measurement.AmountListedOf(chosenLead.Word),
		)
	}

	return performed
}
