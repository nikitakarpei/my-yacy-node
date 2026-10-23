package urlhash_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/urlhash"
)

func TestOneURLHasOneHash(t *testing.T) {
	url := canonicalurltest.CanonicalURLOf(t, "http://host/page")

	if urlhash.Of(url) != urlhash.Of(canonicalurltest.CanonicalURLOf(t, "http://host/page")) {
		t.Fatal("one URL has two hashes")
	}
}

func TestTwoURLsHaveTwoHashes(t *testing.T) {
	first := urlhash.Of(canonicalurltest.CanonicalURLOf(t, "http://host/one"))
	second := urlhash.Of(canonicalurltest.CanonicalURLOf(t, "http://host/two"))

	if first == second {
		t.Fatal("two URLs share one hash")
	}
}
