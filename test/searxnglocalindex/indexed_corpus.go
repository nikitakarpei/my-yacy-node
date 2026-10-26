//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/corpustextservice"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
)

const indexingLimit = 60 * time.Second

func awaitIndexedCorpus(
	t *testing.T,
	ctx context.Context,
	corpusText corpustextservice.CorpusText,
) {
	t.Helper()
	ok := pollwait.For(indexingLimit, func() bool {
		return indexedDocuments(t, ctx, corpusText) >= len(scrapedCorpusURLs())
	})
	if !ok {
		t.Fatal("the scraped corpus was never indexed")
	}
}

func indexedDocuments(
	t *testing.T,
	ctx context.Context,
	corpusText corpustextservice.CorpusText,
) int {
	t.Helper()
	documents := 0
	for _, term := range []string{englishSearchTerm, germanSearchTerm} {
		documents += len(corpusText.MatchesFor(t, ctx, term, ""))
	}
	return documents
}
