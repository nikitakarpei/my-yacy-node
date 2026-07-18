package documentsearch

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type termPosting struct {
	documentIdentifier yacymodel.Hash
	occurrences        int
	textPosition       int
}
