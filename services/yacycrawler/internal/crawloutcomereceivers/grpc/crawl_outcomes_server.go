package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	crawlerv1 "github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawler/v1"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

type RedirectResolutions interface {
	ResolvedURLOf(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
	) (canonicalurl.CanonicalURL, error)
}

type DisposedPages interface {
	DisposedPageOf(
		ctx context.Context,
		canonicalURL canonicalurl.CanonicalURL,
	) (disposal.DisposedPage, bool, error)
}

type crawlOutcomesServer struct {
	crawlerv1.UnimplementedCrawlOutcomesServer
	redirectResolutions RedirectResolutions
	disposedPages       DisposedPages
}

func (s *crawlOutcomesServer) ReadPage(
	ctx context.Context,
	request *crawlerv1.ReadPageRequest,
) (*crawlerv1.PageOutcome, error) {
	canonicalURL, err := canonicalurl.CanonicalURLOf(request.GetUrl())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	resolvedURL, err := s.redirectResolutions.ResolvedURLOf(ctx, canonicalURL)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	disposedPage, disposed, err := s.disposedPages.DisposedPageOf(ctx, canonicalURL)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	outcome := &crawlerv1.PageOutcome{
		CanonicalUrl: canonicalURL.String(),
		ResolvedUrl:  resolvedURL.String(),
	}
	if disposed {
		outcome.Disposal = &crawlerv1.PageDisposal{
			Mark:   string(disposedPage.Mark),
			Reason: string(disposedPage.Reason),
		}
	}
	return outcome, nil
}
