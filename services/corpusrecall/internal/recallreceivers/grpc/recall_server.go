package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

var ErrRequestNamesNoRepresentationKind = errors.New(
	"the request names no representation kind",
)

type Recaller interface {
	Recall(
		ctx context.Context,
		url string,
		representationKinds []recall.RepresentationKind,
	) (recall.RecalledPage, error)
}

type recallServer struct {
	corpusrecallv1.UnimplementedRecallServer
	recaller                          Recaller
	servedRepresentationContractForms representationContractForms
}

func newRecallServer(recaller Recaller, corpora []recall.Corpus) (*recallServer, error) {
	servedRepresentationContractForms, err := servedRepresentationContractFormsFor(corpora)
	if err != nil {
		return nil, err
	}
	return &recallServer{
		recaller:                          recaller,
		servedRepresentationContractForms: servedRepresentationContractForms,
	}, nil
}

func (s *recallServer) Recall(
	ctx context.Context,
	request *corpusrecallv1.RecallRequest,
) (*corpusrecallv1.RecallResponse, error) {
	requestedContractKinds := request.GetKinds()
	if len(requestedContractKinds) == 0 {
		return nil, status.Error(
			codes.InvalidArgument, ErrRequestNamesNoRepresentationKind.Error(),
		)
	}
	representationKinds, err := s.servedRepresentationContractForms.representationKindsFrom(
		requestedContractKinds,
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	recalledPage, err := s.recaller.Recall(ctx, request.GetUrl(), representationKinds)
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
	contractRepresentations, err := s.servedRepresentationContractForms.contractRepresentationsFrom(
		recalledPage.Representations,
	)
	if err != nil {
		return nil, err
	}
	unavailable, err := s.servedRepresentationContractForms.contractRepresentationKindsFrom(
		recalledPage.UnavailableKinds,
	)
	if err != nil {
		return nil, err
	}
	return &corpusrecallv1.RecallResponse{
		Representations: contractRepresentations,
		Unavailable:     unavailable,
	}, nil
}
