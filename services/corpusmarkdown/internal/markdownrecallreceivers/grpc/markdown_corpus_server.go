package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
)

type PageMarkdownRecall interface {
	PageOf(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
	) (markdownrecall.RecalledPage, bool, error)
}

type markdownCorpusServer struct {
	corpusmarkdownv1.UnimplementedMarkdownCorpusServer
	recall PageMarkdownRecall
}

func (s *markdownCorpusServer) RecallPage(
	ctx context.Context,
	request *corpusmarkdownv1.RecallPageRequest,
) (*corpusmarkdownv1.RecallPageResponse, error) {
	requestedURL, err := canonicalurl.CanonicalURLOf(request.GetUrl())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	page, held, err := s.recall.PageOf(ctx, requestedURL)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !held {
		return nil, status.Errorf(
			codes.NotFound, "the corpus holds no markdown for %q", requestedURL,
		)
	}
	return &corpusmarkdownv1.RecallPageResponse{
		CanonicalUrl: page.MarkdownURL.String(),
		Markdown:     string(page.Markdown),
		StoredAt:     timestamppb.New(page.StoredAt),
		Version:      page.Version,
	}, nil
}
