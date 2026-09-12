package pagereading

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type readPage struct {
	document yacymodel.URLHash
	outcome  readOutcome
	text     documenttext.DocumentText
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
