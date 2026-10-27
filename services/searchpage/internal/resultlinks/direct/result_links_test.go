package direct_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/resultlinks/direct"
)

func TestTheLinkOfAResultIsItsPage(t *testing.T) {
	const page = "https://example.org/page?a=1&b=2"

	if link := direct.New().LinkFor(page); link != page {
		t.Errorf("link = %q, want %q", link, page)
	}
}
