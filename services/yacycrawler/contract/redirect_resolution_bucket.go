package yacycrawlcontract

import (
	"crypto/sha256"
	"encoding/hex"
)

const RedirectResolutionBucketName = "YACY_REDIRECT_RESOLUTION"

func RedirectResolutionKey(canonicalURL CanonicalURL) string {
	sum := sha256.Sum256([]byte(canonicalURL.String()))
	return hex.EncodeToString(sum[:])
}
