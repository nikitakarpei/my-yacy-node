//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/markdowncorpusclienttest"
)

const absorptionDeadline = 90 * time.Second

func absorbedMarkdownOf(
	t *testing.T,
	ctx context.Context,
	recallAddress string,
	pageURL string,
) string {
	t.Helper()
	client := markdowncorpusclienttest.New(t, recallAddress)
	var markdown string
	stored := pollwait.For(absorptionDeadline, func() bool {
		recalled, statusCode := client.RecallPage(ctx, pageURL)
		markdown = recalled.Markdown
		return statusCode == http.StatusOK
	})
	if !stored {
		t.Fatalf("corpusmarkdown never stored %q within %s", pageURL, absorptionDeadline)
	}
	return markdown
}
