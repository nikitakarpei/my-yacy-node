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

func TestRecallReportsUnavailableForDisposedPage(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	provisionCrawlInfrastructure(t, ctx, js)

	startCrawler(t, ctx, network.Name)
	recallAddr := startCorpusRecall(t, ctx, network.Name)

	client := dialRecall(t, recallAddr)

	callCtx, cancel := context.WithTimeout(ctx, recallDeadline)
	defer cancel()
	start := time.Now()
	resp := recall(t, callCtx, client, originURL+"missing")
	elapsed := time.Since(start)

	if elapsed >= corpusRecallDeadline {
		t.Errorf(
			"recall took %s, want an early return well inside the %s deadline",
			elapsed,
			corpusRecallDeadline,
		)
	}

	if reps := resp.GetRepresentations(); len(reps) != 0 {
		t.Errorf("representations = %v, want none for a disposed page", reps)
	}
	want := []corpusrecallv1.RepresentationKind{
		corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
	}
	if got := resp.GetUnavailable(); !equalKinds(got, want) {
		t.Errorf("unavailable = %v, want %v", got, want)
	}
}

func equalKinds(got, want []corpusrecallv1.RepresentationKind) bool {
	if len(got) != len(want) {
		return false
	}
	for i, kind := range want {
		if got[i] != kind {
			return false
		}
	}
	return true
}
