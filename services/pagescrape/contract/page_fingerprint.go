package pagescrapecontract

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

func pageFingerprintOf(pageURL canonicalurl.CanonicalURL) string {
	sum := sha256.Sum256([]byte(pageURL.String()))
	return hex.EncodeToString(sum[:])
}
