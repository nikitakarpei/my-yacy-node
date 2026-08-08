//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
)

const indexingLimit = 60 * time.Second

func awaitIndexedCorpus(t *testing.T, documents func() int) {
	t.Helper()
	ok := pollwait.For(indexingLimit, func() bool {
		return documents() >= len(crawledPages())
	})
	if !ok {
		t.Fatal("the crawled corpus was never indexed")
	}
}

func assertCatchAllHoldsTheUnconfiguredPage(t *testing.T, documents int) {
	t.Helper()
	if documents != 1 {
		t.Errorf("catch-all index holds %d documents, want 1", documents)
	}
}
