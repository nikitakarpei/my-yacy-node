package yacycrawlcontract

import (
	"crypto/sha256"
	"encoding/hex"
)

const DisposedPagesBucketName = "YACY_DISPOSED_PAGES"

func DisposedPageKey(canonicalURL CanonicalURL) string {
	sum := sha256.Sum256([]byte(canonicalURL.String()))
	return hex.EncodeToString(sum[:])
}
