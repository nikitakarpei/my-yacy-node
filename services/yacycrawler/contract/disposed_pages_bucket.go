package yacycrawlcontract

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const DisposedPagesBucketName = "YACY_DISPOSED_PAGES"

func DisposedPageKey(canonicalURL canonicalurl.CanonicalURL) string {
	sum := sha256.Sum256([]byte(canonicalURL.String()))
	return hex.EncodeToString(sum[:])
}
