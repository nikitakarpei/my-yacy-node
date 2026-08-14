package grpc

import (
	"errors"
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

var ErrRepresentationKindHasNoContractForm = errors.New(
	"no contract form serves this representation kind",
)

var ErrRepresentationDoesNotMatchItsKind = errors.New(
	"representation does not match its kind",
)

type representationContractForm struct {
	representationKind         recall.RepresentationKind
	contractRepresentationKind corpusrecallv1.RepresentationKind
	contractRepresentationFrom func(recall.Representation) (*corpusrecallv1.Representation, bool)
}

type representationContractForms []representationContractForm

func servedRepresentationContractFormsFor(
	corpora []recall.Corpus,
) (representationContractForms, error) {
	served := make(representationContractForms, 0, len(corpora))
	for _, corpus := range corpora {
		contractForm, hasContractForm := allRepresentationContractForms().
			contractFormOfRepresentationKind(corpus.RepresentationKind())
		if !hasContractForm {
			return nil, noContractFormForRepresentationKind(corpus.RepresentationKind())
		}
		served = append(served, contractForm)
	}
	return served, nil
}

func (contractForms representationContractForms) contractFormOfRepresentationKind(
	representationKind recall.RepresentationKind,
) (representationContractForm, bool) {
	for _, contractForm := range contractForms {
		if contractForm.representationKind == representationKind {
			return contractForm, true
		}
	}
	return representationContractForm{}, false
}

func allRepresentationContractForms() representationContractForms {
	return representationContractForms{markdownRepresentationContractForm()}
}

func noContractFormForRepresentationKind(representationKind recall.RepresentationKind) error {
	return fmt.Errorf("%w: %s", ErrRepresentationKindHasNoContractForm, representationKind)
}

func (contractForms representationContractForms) representationKindsFrom(
	contractRepresentationKinds []corpusrecallv1.RepresentationKind,
) ([]recall.RepresentationKind, error) {
	representationKinds := make([]recall.RepresentationKind, 0, len(contractRepresentationKinds))
	for _, contractRepresentationKind := range contractRepresentationKinds {
		representationKind, served := contractForms.representationKindFrom(
			contractRepresentationKind,
		)
		if !served {
			return nil, fmt.Errorf("unserved representation kind %s", contractRepresentationKind)
		}
		representationKinds = append(representationKinds, representationKind)
	}
	return representationKinds, nil
}

func (contractForms representationContractForms) representationKindFrom(
	contractRepresentationKind corpusrecallv1.RepresentationKind,
) (recall.RepresentationKind, bool) {
	for _, contractForm := range contractForms {
		if contractForm.contractRepresentationKind == contractRepresentationKind {
			return contractForm.representationKind, true
		}
	}
	return "", false
}

func (contractForms representationContractForms) contractRepresentationsFrom(
	recalledRepresentations []recall.RecalledRepresentation,
) ([]*corpusrecallv1.Representation, error) {
	contractRepresentations := make(
		[]*corpusrecallv1.Representation, 0, len(recalledRepresentations),
	)
	for _, recalled := range recalledRepresentations {
		expressed, err := contractForms.contractRepresentationFrom(recalled)
		if err != nil {
			return nil, err
		}
		contractRepresentations = append(contractRepresentations, expressed)
	}
	return contractRepresentations, nil
}

func (contractForms representationContractForms) contractRepresentationFrom(
	recalled recall.RecalledRepresentation,
) (*corpusrecallv1.Representation, error) {
	contractForm, served := contractForms.contractFormOfRepresentationKind(recalled.Kind)
	if !served {
		return nil, noContractFormForRepresentationKind(recalled.Kind)
	}
	expressed, expressible := contractForm.contractRepresentationFrom(recalled.Representation)
	if !expressible {
		return nil, representationNotMatchingItsKind(recalled)
	}
	return expressed, nil
}

func representationNotMatchingItsKind(recalled recall.RecalledRepresentation) error {
	return fmt.Errorf(
		"%w: %s holds %T",
		ErrRepresentationDoesNotMatchItsKind, recalled.Kind, recalled.Representation,
	)
}

func (contractForms representationContractForms) contractRepresentationKindsFrom(
	representationKinds []recall.RepresentationKind,
) []corpusrecallv1.RepresentationKind {
	contractRepresentationKinds := make(
		[]corpusrecallv1.RepresentationKind,
		0,
		len(representationKinds),
	)
	for _, contractForm := range contractForms {
		if slices.Contains(representationKinds, contractForm.representationKind) {
			contractRepresentationKinds = append(
				contractRepresentationKinds,
				contractForm.contractRepresentationKind,
			)
		}
	}
	return contractRepresentationKinds
}
