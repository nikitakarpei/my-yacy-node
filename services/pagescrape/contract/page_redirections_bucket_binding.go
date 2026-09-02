package pagescrapecontract

import "github.com/nikitakarpei/yacy-rwi-node/canonicalurl"

const PageRedirectionsBucketName = "YACY_PAGE_REDIRECTIONS"

func PageRedirectionKeyOf(requestedURL canonicalurl.CanonicalURL) string {
	return pageFingerprintOf(requestedURL)
}
