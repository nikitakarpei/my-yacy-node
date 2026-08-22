// Package canonicalurltest builds the canonical URL of a raw URL in a test.
package canonicalurltest

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

func CanonicalURLOf(t *testing.T, rawURL string) canonicalurl.CanonicalURL {
	t.Helper()
	canonical, err := canonicalurl.CanonicalURLOf(rawURL)
	if err != nil {
		t.Fatalf("canonical url of %q: %v", rawURL, err)
	}
	return canonical
}
