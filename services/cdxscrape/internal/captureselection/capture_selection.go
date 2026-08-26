// Package captureselection holds the rule that says which of the captures an archive
// lists earn a scrape request. An archive keeps every capture of a page it ever took; a
// corpus wants the page as it was last seen.
package captureselection

import (
	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/cdxindex"
)

func NewestCapturesOf(captures []cdxindex.Capture) []cdxindex.Capture {
	newestCaptures := []cdxindex.Capture{}
	positionOfURLKey := map[string]int{}
	for _, capture := range captures {
		position, seen := positionOfURLKey[capture.URLKey]
		if !seen {
			positionOfURLKey[capture.URLKey] = len(newestCaptures)
			newestCaptures = append(newestCaptures, capture)
			continue
		}
		if capture.Timestamp > newestCaptures[position].Timestamp {
			newestCaptures[position] = capture
		}
	}
	return newestCaptures
}
