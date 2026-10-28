package documentasks

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type EndedListings struct {
	Word                         yacymodel.Hash
	TimeSinceTheInquiryBegan     time.Duration
	AmountOfDocumentsListedSoFar int
}
