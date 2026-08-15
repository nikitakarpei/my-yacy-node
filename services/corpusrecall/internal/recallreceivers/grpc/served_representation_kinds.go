package grpc

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

var ErrRepresentationKindNotInTheContract = errors.New(
	"the contract has no kind for this representation kind",
)

var ErrContractRepresentationKindNotServed = errors.New(
	"no served representation kind answers this contract kind",
)

type servedRepresentationKinds []recall.RepresentationKind

func servedRepresentationKindsFor(corpora []recall.Corpus) (servedRepresentationKinds, error) {
	served := make(servedRepresentationKinds, 0, len(corpora))
	for _, corpus := range corpora {
		representationKind := corpus.RepresentationKind()
		if _, inContract := contractRepresentationKindOf(representationKind); !inContract {
			return nil, representationKindNotInTheContract(representationKind)
		}
		served = append(served, representationKind)
	}
	return served, nil
}

func contractRepresentationKindOf(
	representationKind recall.RepresentationKind,
) (corpusrecallv1.RepresentationKind, bool) {
	switch representationKind {
	case markdown.Kind:
		return corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN, true
	default:
		return corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_UNSPECIFIED, false
	}
}

func representationKindNotInTheContract(representationKind recall.RepresentationKind) error {
	return fmt.Errorf("%w: %s", ErrRepresentationKindNotInTheContract, representationKind)
}

func (served servedRepresentationKinds) representationKindsFrom(
	contractRepresentationKinds []corpusrecallv1.RepresentationKind,
) ([]recall.RepresentationKind, error) {
	representationKinds := make([]recall.RepresentationKind, 0, len(contractRepresentationKinds))
	for _, contractRepresentationKind := range contractRepresentationKinds {
		representationKind, isServed := served.representationKindFrom(contractRepresentationKind)
		if !isServed {
			return nil, contractRepresentationKindNotServed(contractRepresentationKind)
		}
		representationKinds = append(representationKinds, representationKind)
	}
	return representationKinds, nil
}

func (served servedRepresentationKinds) representationKindFrom(
	contractRepresentationKind corpusrecallv1.RepresentationKind,
) (recall.RepresentationKind, bool) {
	for _, representationKind := range served {
		contractRepresentationKindOfServed, inContract := contractRepresentationKindOf(
			representationKind,
		)
		if inContract && contractRepresentationKindOfServed == contractRepresentationKind {
			return representationKind, true
		}
	}
	return "", false
}

func contractRepresentationKindNotServed(
	contractRepresentationKind corpusrecallv1.RepresentationKind,
) error {
	return fmt.Errorf("%w: %s", ErrContractRepresentationKindNotServed, contractRepresentationKind)
}

func contractRepresentationKindsFrom(
	representationKinds []recall.RepresentationKind,
) ([]corpusrecallv1.RepresentationKind, error) {
	contractRepresentationKinds := make(
		[]corpusrecallv1.RepresentationKind, 0, len(representationKinds),
	)
	for _, representationKind := range representationKinds {
		contractRepresentationKind, inContract := contractRepresentationKindOf(representationKind)
		if !inContract {
			return nil, representationKindNotInTheContract(representationKind)
		}
		contractRepresentationKinds = append(
			contractRepresentationKinds, contractRepresentationKind,
		)
	}
	return contractRepresentationKinds, nil
}
