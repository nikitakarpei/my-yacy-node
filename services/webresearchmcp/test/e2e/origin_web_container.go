//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/staticpage"
)

const (
	originAlias        = "origin"
	originCanonicalURL = "http://" + originAlias + "/"
	originTitle        = "Research subject"
	originParagraph    = "The origin serves one page of prose about the research subject. "
	originParagraphs   = 12
)

func startOrigin(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	staticpage.Start(t, ctx, networkName, originAlias, originPage())
}

func originPage() string {
	return `<html lang="en"><title>` + originTitle + `</title><body><p>` +
		strings.Repeat(originParagraph, originParagraphs) +
		`</p></body></html>`
}
