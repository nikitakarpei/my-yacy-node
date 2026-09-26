package pagereading

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type pageReadResult struct {
	document          yacymodel.URLHash
	outcome           readOutcome
	pageContents      pagecontents.PageContents
	timeSpentFetching time.Duration
	timeSpentReading  time.Duration
}

type readOutcome int

const (
	pageWasRead readOutcome = iota
	pageWasUnreachable
	pageWasRefused
	pageWasGone
	pageRefusesIndexing
	pageWasUnreadable
	pageWasOfAnUnsupportedKind
	pageWasOutOfBudget
	pageWasCutOff
)
