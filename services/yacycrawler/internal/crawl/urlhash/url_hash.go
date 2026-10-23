// Package urlhash names a page URL in crawl state by a fixed-size digest of its
// canonical form, so a record key stays short whatever the URL's length.
package urlhash

import (
	"crypto/sha256"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type URLHash [16]byte

func Of(url canonicalurl.CanonicalURL) URLHash {
	digest := sha256.Sum256([]byte(url.String()))
	return URLHash(digest[:len(URLHash{})])
}
