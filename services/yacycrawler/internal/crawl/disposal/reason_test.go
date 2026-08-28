package disposal_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

func TestNotDisposedDisposesOfNothing(t *testing.T) {
	if disposal.NotDisposed.DisposedThePage() {
		t.Fatal("a published page carries no disposal")
	}
}

func TestANamedReasonDisposes(t *testing.T) {
	if !disposal.IndexingRefused.DisposedThePage() {
		t.Fatalf("%q should count as a disposal", disposal.IndexingRefused)
	}
}
