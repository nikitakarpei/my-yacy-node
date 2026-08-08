package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

type Recaller interface {
	Recall(
		ctx context.Context,
		url string,
		kinds []recall.RepresentationKind,
	) (recall.RecalledPage, error)
}

type recallServer struct {
	corpusrecallv1.UnimplementedRecallServer
	recaller Recaller
	forms    servedRepresentationForms
}

func newRecallServer(recaller Recaller, corpora []recall.Corpus) (*recallServer, error) {
	forms, err := servedRepresentationFormsFor(corpora)
	if err != nil {
		return nil, err
	}
	return &recallServer{recaller: recaller, forms: forms}, nil
}

func (s *recallServer) Recall(
	ctx context.Context,
	request *corpusrecallv1.RecallRequest,
) (*corpusrecallv1.RecallResponse, error) {
	requestedContractKinds := request.GetKinds()
	kinds, err := s.forms.representationKindsFrom(requestedContractKinds)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	recalledPage, err := s.recaller.Recall(ctx, request.GetUrl(), kinds)
	if err != nil {
		if errors.Is(err, recall.ErrTooManyRequestsInFlight) {
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	response, err := s.recallResponseFrom(recalledPage)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return response, nil
}

func (s *recallServer) recallResponseFrom(
	recalledPage recall.RecalledPage,
) (*corpusrecallv1.RecallResponse, error) {
	contractRepresentations, err := s.forms.contractRepresentationsFrom(
		recalledPage.Representations,
	)
	if err != nil {
		return nil, err
	}
	return &corpusrecallv1.RecallResponse{
		Representations: contractRepresentations,
		Unavailable:     s.forms.contractRepresentationKindsFrom(recalledPage.UnavailableKinds),
	}, nil
}
