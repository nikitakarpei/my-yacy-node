//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/corpustextservice"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pagescrapeservice"
)

func TestEveryCorpusAbsorbsTheSameOfferedPageEndToEnd(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	pagescrapeservice.Start(t, ctx, network.Name)
	startManticore(t, ctx, network.Name)
	recallAddress := startCorpusMarkdown(t, ctx, network.Name)
	corpusText := corpustextservice.Start(
		t, ctx, network.Name, natsURL, manticoreCorpusTextEnv(),
	)

	js := connectJetStream(t, natsURL)
	publishScrapeRequest(t, ctx, js, originURL)

	markdown := absorbedMarkdownOf(t, ctx, recallAddress, originURL)
	if !strings.Contains(markdown, originBody) {
		t.Errorf("absorbed markdown = %q, want it to contain %q", markdown, originBody)
	}

	text := corpusText.FirstMatchFor(t, ctx, originBody, indexedLanguage)
	if text.Title != originTitle {
		t.Errorf("absorbed title = %q, want %q", text.Title, originTitle)
	}
	if !strings.Contains(text.Snippet, originBody) {
		t.Errorf("absorbed text = %q, want it to contain %q", text.Snippet, originBody)
	}
}
