package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
	markdowncorporagrpc "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/markdowncorpora/grpc"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
)

const (
	pageAddress    = "https://example.org/page"
	heldMarkdown   = "# Research subject"
	heldVersion    = "version-1"
	recallDeadline = 5 * time.Second
)

type corpusServing struct {
	corpusmarkdownv1.UnimplementedMarkdownCorpusServer
	held     *corpusmarkdownv1.RecallPageResponse
	notFound bool
}

func (s *corpusServing) RecallPage(
	_ context.Context,
	_ *corpusmarkdownv1.RecallPageRequest,
) (*corpusmarkdownv1.RecallPageResponse, error) {
	if s.notFound {
		return nil, status.Error(codes.NotFound, "the corpus holds no markdown for the page")
	}
	if s.held == nil {
		return nil, status.Error(codes.Unavailable, "the corpus is away")
	}
	return s.held, nil
}

func openCorpusUnderTest(t *testing.T, serving *corpusServing) *markdowncorporagrpc.MarkdownCorpus {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	endpoint := grpcserver.NewServer()
	corpusmarkdownv1.RegisterMarkdownCorpusServer(endpoint, serving)
	go func() { _ = endpoint.Serve(listener) }()
	t.Cleanup(endpoint.Stop)

	corpus, err := markdowncorporagrpc.OpenMarkdownCorpus(listener.Addr().String(), recallDeadline)
	if err != nil {
		t.Fatalf("open the markdown corpus: %v", err)
	}
	t.Cleanup(func() { _ = corpus.Close() })
	return corpus
}

func TestPageTheCorpusHoldsIsReadWithItsVersionAndTheTimeItWasStored(t *testing.T) {
	storedAt := time.Now().UTC().Truncate(time.Second)
	corpus := openCorpusUnderTest(t, &corpusServing{
		held: &corpusmarkdownv1.RecallPageResponse{
			CanonicalUrl: pageAddress,
			Markdown:     heldMarkdown,
			Version:      heldVersion,
			StoredAt:     timestamppb.New(storedAt),
		},
	})

	held, err := corpus.PageMarkdownAt(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if held.Markdown != heldMarkdown {
		t.Errorf("markdown = %q, want %q", held.Markdown, heldMarkdown)
	}
	if held.Version != heldVersion {
		t.Errorf("version = %q, want %q", held.Version, heldVersion)
	}
	if !held.StoredAt.Equal(storedAt) {
		t.Errorf("stored at = %v, want %v", held.StoredAt, storedAt)
	}
}

func TestPageTheCorpusDoesNotHoldIsAnsweredAsNotInTheCorpus(t *testing.T) {
	corpus := openCorpusUnderTest(t, &corpusServing{notFound: true})

	_, err := corpus.PageMarkdownAt(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	)
	if !errors.Is(err, pageread.ErrPageNotInCorpus) {
		t.Fatalf("error = %v, want %v", err, pageread.ErrPageNotInCorpus)
	}
}

func TestCorpusThatIsAwayFailsTheRead(t *testing.T) {
	corpus := openCorpusUnderTest(t, &corpusServing{})

	if _, err := corpus.PageMarkdownAt(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	); err == nil {
		t.Fatal("the read answered without an error, want the corpus that is away to fail")
	}
}
