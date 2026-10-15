package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPostingPageVersionIsItsLastModifiedDay(t *testing.T) {
	posting := yacymodel.RWIPosting{LastModified: 20000}

	if got := posting.PageVersion(); got != posting.LastModified {
		t.Fatalf("PageVersion() = %d, want %d", got, posting.LastModified)
	}
}
