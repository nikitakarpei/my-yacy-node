package queryanswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
)

type PageContents struct {
	Text       documenttext.DocumentText
	LinkCounts LinkCounts
}
