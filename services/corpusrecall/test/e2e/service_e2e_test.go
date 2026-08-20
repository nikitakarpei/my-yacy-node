//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/recallclienttest"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
)

const recallServeLimit = 90 * time.Second

func TestRecallServesCrawledMarkdown(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	crawlNATSURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)

	startCrawler(t, ctx, network.Name)
	startCorpusMarkdown(t, ctx, network.Name)
	awaitRecallPreconditions(t, ctx, connectJetStream(t, crawlNATSURL))
	recallAddr := startCorpusRecall(t, ctx, network.Name)

	client := recallclienttest.New(t, recallAddr)

	var resp *corpusrecallv1.RecallResponse
	served := pollwait.For(recallServeLimit, func() bool {
		callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		resp = recall(t, callCtx, client, originURL)
		return len(resp.GetRepresentations()) == 1
	})
	if !served {
		t.Fatalf("recall never returned markdown within %s", recallServeLimit)
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

	crawlNATSURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)

	startCrawler(t, ctx, network.Name)
	startCorpusMarkdown(t, ctx, network.Name)
	awaitRecallPreconditions(t, ctx, connectJetStream(t, crawlNATSURL))
	recallAddr := startCorpusRecall(t, ctx, network.Name)

	client := recallclienttest.New(t, recallAddr)

	callCtx, cancel := context.WithTimeout(ctx, recallServeLimit)
	defer cancel()
	start := time.Now()
	resp := recall(t, callCtx, client, originURL+"missing")
	elapsed := time.Since(start)

	if elapsed >= corpusRecallLimit {
		t.Errorf(
			"recall took %s, want an early return well inside the %s recall limit",
			elapsed,
			corpusRecallLimit,
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

func recall(
	t *testing.T,
	ctx context.Context,
	client corpusrecallv1.RecallClient,
	url string,
) *corpusrecallv1.RecallResponse {
	t.Helper()
	resp, err := client.Recall(ctx, &corpusrecallv1.RecallRequest{
		Url: url,
		Kinds: []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
		},
	})
	if err != nil {
		t.Fatalf("recall %q: %v", url, err)
	}
	return resp
}
