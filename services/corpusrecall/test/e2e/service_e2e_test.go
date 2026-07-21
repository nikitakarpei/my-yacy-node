//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
)

const recallDeadline = 90 * time.Second

func TestRecallServesCrawledMarkdown(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	provisionCrawlInfrastructure(t, ctx, js)

	startCrawler(t, ctx, network.Name)
	startCorpusMarkdown(t, ctx, network.Name)
	recallAddr := startCorpusRecall(t, ctx, network.Name)

	client := dialRecall(t, recallAddr)

	var resp *corpusrecallv1.RecallResponse
	served := pollwait.For(recallDeadline, func() bool {
		callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		resp = recall(t, callCtx, client, originURL)
		return len(resp.GetRepresentations()) == 1
	})
	if !served {
		t.Fatalf("recall never returned markdown within %s", recallDeadline)
	}

	markdown := resp.GetRepresentations()[0].GetMarkdown()
	if markdown == nil {
		t.Fatalf("representation is not markdown: %v", resp.GetRepresentations()[0])
	}
	if !strings.Contains(markdown.GetMarkdown(), originBody) {
		t.Errorf(
			"recalled markdown = %q, want it to contain %q",
			markdown.GetMarkdown(),
			originBody,
		)
	}

	if u := resp.GetUnavailable(); len(u) != 0 {
		t.Errorf("unavailable = %v, want none", u)
	}
}
