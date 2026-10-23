package markdownrecall_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
)

func TestVersionOfStaysTheSameForTheSameMarkdown(t *testing.T) {
	first := markdownrecall.VersionOf([]byte("# Hi"))
	second := markdownrecall.VersionOf([]byte("# Hi"))

	if first != second {
		t.Errorf("version = %q for the same markdown, want the earlier %q", second, first)
	}
}

func TestVersionOfChangesWithTheMarkdown(t *testing.T) {
	if markdownrecall.VersionOf([]byte("# Hi")) == markdownrecall.VersionOf([]byte("# Hi again")) {
		t.Error("two different markdowns share one version")
	}
}
