// Package urlhashtest builds the URL hash of an address in a test.
package urlhashtest

import (
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func URLHashOf(tb testing.TB, address string) yacymodel.URLHash {
	tb.Helper()

	webAddress, err := url.Parse(address)
	if err != nil {
		tb.Fatalf("parse %q: %v", address, err)
	}

	return yacymodel.URLHashOf(webAddress)
}
