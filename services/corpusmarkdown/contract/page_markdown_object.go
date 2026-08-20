// Package pagemarkdownstore is the single source of truth binding a crawled
// page's canonical URL to the bucket and object name that hold its markdown,
// shared by the writer that fills the corpus and the readers that recall from it.
package pagemarkdownstore

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const BucketName = "YACY_PAGE_MARKDOWN"

func ObjectName(canonicalURL yacycrawlcontract.CanonicalURL) string {
	sum := sha256.Sum256([]byte(canonicalURL.String()))
	return hex.EncodeToString(sum[:])
}
