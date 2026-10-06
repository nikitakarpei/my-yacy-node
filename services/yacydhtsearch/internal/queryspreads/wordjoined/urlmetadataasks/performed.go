package urlmetadataasks

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Performed struct {
	AmountOfLookedUpDocuments             int
	AmountOfLookedUpDocumentsWithMetadata int
	AmountOfDocumentsNotAsked             int
	EndReason                             EndReason
	AmountOfDocumentsCutOff               int
	AmountOfDocumentsPerAsk               []int
	TimeToFirstAsk                        yacymodel.Optional[time.Duration]
}
