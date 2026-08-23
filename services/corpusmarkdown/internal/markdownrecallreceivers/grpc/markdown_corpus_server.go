package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
)

type PageMarkdownCorpus interface {
	MarkdownOf(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) ([]byte, bool, error)
}

type markdownCorpusServer struct {
	corpusmarkdownv1.UnimplementedMarkdownCorpusServer
	corpus PageMarkdownCorpus
}

func (s *markdownCorpusServer) RecallPage(
	ctx context.Context,
	request *corpusmarkdownv1.RecallPageRequest,
) (*corpusmarkdownv1.RecallPageResponse, error) {
	canonicalURL, err := canonicalurl.CanonicalURLOf(request.GetUrl())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	markdown, held, err := s.corpus.MarkdownOf(ctx, canonicalURL)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !held {
		return nil, status.Errorf(
			codes.NotFound, "the corpus holds no markdown for %q", canonicalURL,
		)
	}
	return &corpusmarkdownv1.RecallPageResponse{
		CanonicalUrl: canonicalURL.String(),
		Markdown:     string(markdown),
	}, nil
}
