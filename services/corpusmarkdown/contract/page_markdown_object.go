// Package pagemarkdownstore is the single source of truth binding a crawled page's
// canonical URL to the object name that holds its markdown, and to the path and answer
// body a reader recalls that markdown through, shared by the writer that fills the corpus
// and the readers that recall from it.
package pagemarkdownstore

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

func ObjectNameOf(canonicalURL canonicalurl.CanonicalURL) string {
	return fingerprintOf(canonicalURL)
}

func fingerprintOf(canonicalURL canonicalurl.CanonicalURL) string {
	sum := sha256.Sum256([]byte(canonicalURL.String()))
	return hex.EncodeToString(sum[:])
}
