package recallgrpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerecall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

type Recaller interface {
	Recall(ctx context.Context, url string, kinds []pagerecall.Kind) (pagerecall.Result, error)
}

type RecallServer struct {
	corpusrecallv1.UnimplementedRecallServer
	recaller    Recaller
	translation representationTranslation
}

func NewRecallServer(recaller Recaller, codecs []RepresentationCodec) *RecallServer {
	return &RecallServer{recaller: recaller, translation: newRepresentationTranslation(codecs)}
}

func (s *RecallServer) Recall(
	ctx context.Context,
	request *corpusrecallv1.RecallRequest,
) (*corpusrecallv1.RecallResponse, error) {
	kinds, err := s.translation.requestedKinds(request.GetKinds())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	result, err := s.recaller.Recall(ctx, request.GetUrl(), kinds)
	if err != nil {
		if errors.Is(err, pagerecall.ErrTooManyInFlight) {
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return s.translation.recallResponse(result), nil
}
