package pagereading

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type readPage struct {
	document yacymodel.URLHash
	outcome  readOutcome
	text     PageText
}

type readOutcome int

const (
	pageRead readOutcome = iota
	pageUnreachable
	pageRefused
	pageUnreadable
	pageOfAnUnsupportedKind
	pageOutOfBudget
)
