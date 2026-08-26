// Package capturereplay owns the address an archive replays a capture at. The address
// carries the collection and the moment of capture, and ends with the url the capture was
// taken from. That url is escaped whole, so that the slashes, the dots, and the query
// inside it stay part of it and reach the archive as they were captured.
//
// The moment of capture carries the mp_ modifier, which asks the archive for the page
// itself rather than for the frame it shows a reader, and for links and subresources that
// point back into the archive rather than at the origin.
package capturereplay

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/cdxscrape/internal/cdxindex"
)

const archivedPageModifier = "mp_"

type Archive struct {
	archiveURL string
	collection string
}

func New(archiveURL *url.URL, collection string) *Archive {
	return &Archive{
		archiveURL: strings.TrimSuffix(archiveURL.String(), "/"),
		collection: collection,
	}
}

func (a *Archive) ReplayURLOf(capture cdxindex.Capture) (canonicalurl.CanonicalURL, error) {
	replayURL := strings.Join(
		[]string{
			a.archiveURL,
			a.collection,
			capture.Timestamp + archivedPageModifier,
			url.PathEscape(capture.OriginalURL),
		},
		"/",
	)
	canonicalReplayURL, err := canonicalurl.CanonicalURLOf(replayURL)
	if err != nil {
		return canonicalurl.CanonicalURL{}, fmt.Errorf("read replay url %s: %w", replayURL, err)
	}
	return canonicalReplayURL, nil
}
