// Package pagemarkdownstore is the single source of truth binding a crawled
// page's canonical URL to the bucket and object name that hold its markdown,
// shared by the writer that fills the corpus and the readers that recall from it.
package pagemarkdownstore

import (
	"crypto/sha256"
	"encoding/hex"
)

const BucketName = "YACY_PAGE_MARKDOWN"

func ObjectName(canonicalURL string) string {
	sum := sha256.Sum256([]byte(canonicalURL))
	return hex.EncodeToString(sum[:])
}
