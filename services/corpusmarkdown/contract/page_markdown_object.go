// Package pagemarkdownstore is the single source of truth binding a crawled page's
// canonical URL to the bucket and object name that hold its markdown, and to the subject
// that carries the outcome of scraping it, shared by the writer that fills the corpus and
// the readers and waiters that recall from it.
package pagemarkdownstore

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const BucketName = "YACY_PAGE_MARKDOWN"

func ObjectNameOf(canonicalURL canonicalurl.CanonicalURL) string {
	return fingerprintOf(canonicalURL)
}

func fingerprintOf(canonicalURL canonicalurl.CanonicalURL) string {
	sum := sha256.Sum256([]byte(canonicalURL.String()))
	return hex.EncodeToString(sum[:])
}
