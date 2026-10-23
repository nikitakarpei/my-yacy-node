//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/corpustextservice"
)

func awaitBleveCorpus(
	t *testing.T,
	ctx context.Context,
	corpusText corpustextservice.CorpusText,
) {
	t.Helper()
	awaitIndexedCorpus(t, func() int {
		documents := 0
		for _, term := range []string{englishSearchTerm, germanSearchTerm, spanishSearchTerm} {
			documents += len(corpusText.MatchesFor(t, ctx, term, ""))
		}
		return documents
	})
}
