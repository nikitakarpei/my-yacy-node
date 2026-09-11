// Package peerorder orders the answered items the way the answering peers put
// them: one item of each peer per round, and one item per document.
package peerorder

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
)

type Ordering struct{}

func (Ordering) OrderedItemsOf(
	answers peeranswers.AnsweredQuery,
) []peeranswers.AnsweredItem {
	return answers.ItemOfEachAnsweredDocument()
}
