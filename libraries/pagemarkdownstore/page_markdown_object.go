// Package pagemarkdownstore is the single source of truth binding a crawled
// page's canonical URL to its markdown object in the NATS JetStream Object
// Store, shared by the writer that fills the store and future readers.
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
