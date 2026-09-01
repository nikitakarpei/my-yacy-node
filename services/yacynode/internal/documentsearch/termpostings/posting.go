package termpostings

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type Posting struct {
	DocumentHash yacymodel.URLHash
	Occurrences  int
	TextPosition int
}
