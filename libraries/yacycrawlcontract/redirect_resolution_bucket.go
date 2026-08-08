package yacycrawlcontract

import (
	"crypto/sha256"
	"encoding/hex"
)

const RedirectResolutionBucketName = "YACY_REDIRECT_RESOLUTION"

func RedirectResolutionKey(canonicalURL string) string {
	sum := sha256.Sum256([]byte(canonicalURL))
	return hex.EncodeToString(sum[:])
}
