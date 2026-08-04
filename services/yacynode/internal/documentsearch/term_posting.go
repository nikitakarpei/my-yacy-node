package documentsearch

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type termPosting struct {
	documentHash yacymodel.URLHash
	occurrences  int
	textPosition int
}
