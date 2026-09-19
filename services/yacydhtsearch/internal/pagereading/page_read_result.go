package pagereading

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type pageReadResult struct {
	document          yacymodel.URLHash
	outcome           readOutcome
	pageContents      queryanswers.PageContents
	timeSpentFetching time.Duration
	timeSpentReading  time.Duration
}

type readOutcome int

const (
	pageWasRead readOutcome = iota
	pageWasUnreachable
	pageWasRefused
	pageWasGone
	pageWasUnreadable
	pageWasOfAnUnsupportedKind
	pageWasOutOfBudget
)
