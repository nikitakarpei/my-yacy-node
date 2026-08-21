package yacycrawlcontract

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const RedirectResolutionBucketName = "YACY_REDIRECT_RESOLUTION"

func RedirectResolutionKey(canonicalURL canonicalurl.CanonicalURL) string {
	sum := sha256.Sum256([]byte(canonicalURL.String()))
	return hex.EncodeToString(sum[:])
}
