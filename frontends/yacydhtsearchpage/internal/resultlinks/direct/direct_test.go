package direct_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/resultlinks/direct"
)

func TestLinkForIsTheAddress(t *testing.T) {
	if link := (direct.Direct{}).LinkFor("https://yacy.net/"); link != "https://yacy.net/" {
		t.Errorf("LinkFor = %q, want the address", link)
	}
}
