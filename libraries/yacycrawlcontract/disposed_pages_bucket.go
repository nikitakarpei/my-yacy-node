package yacycrawlcontract

import (
	"crypto/sha256"
	"encoding/hex"
)

const DisposedPagesBucketName = "YACY_DISPOSED_PAGES"

func DisposedPageKey(canonicalURL string) string {
	sum := sha256.Sum256([]byte(canonicalURL))
	return hex.EncodeToString(sum[:])
}
