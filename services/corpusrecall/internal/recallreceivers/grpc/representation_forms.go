package grpc

import (
	"errors"
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

var ErrRepresentationKindNotInContract = errors.New(
	"representation kind has no form in the recall contract",
)

type representationForm struct {
	kind                       recall.RepresentationKind
	contractKind               corpusrecallv1.RepresentationKind
	contractRepresentationFrom func(recall.Representation) (*corpusrecallv1.Representation, bool)
}

type servedRepresentationForms []representationForm

func servedRepresentationFormsFor(
	corpora []recall.Corpus,
) (servedRepresentationForms, error) {
	served := make(servedRepresentationForms, 0, len(corpora))
	for _, corpus := range corpora {
		form, formed := representationFormOfKind(corpus.RepresentationKind())
		if !formed {
			return nil, kindNotInContract(corpus.RepresentationKind())
		}
		served = append(served, form)
	}
	return served, nil
}

func representationFormOfKind(
	kind recall.RepresentationKind,
) (representationForm, bool) {
	for _, form := range []representationForm{markdownRepresentationForm()} {
		if form.kind == kind {
			return form, true
		}
	}
	return representationForm{}, false
}

func kindNotInContract(kind recall.RepresentationKind) error {
	return fmt.Errorf("%w: %s", ErrRepresentationKindNotInContract, kind)
}

func (forms servedRepresentationForms) representationKindsFrom(
	contractKinds []corpusrecallv1.RepresentationKind,
) ([]recall.RepresentationKind, error) {
	kinds := make([]recall.RepresentationKind, 0, len(contractKinds))
	for _, contractKind := range contractKinds {
		kind, served := forms.representationKindFrom(contractKind)
		if !served {
			return nil, fmt.Errorf("unserved representation kind %s", contractKind)
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}

func (forms servedRepresentationForms) representationKindFrom(
	contractKind corpusrecallv1.RepresentationKind,
) (recall.RepresentationKind, bool) {
	for _, form := range forms {
		if form.contractKind == contractKind {
			return form.kind, true
		}
	}
	return "", false
}

func (forms servedRepresentationForms) contractRepresentationsFrom(
	recalledRepresentations []recall.RecalledRepresentation,
) ([]*corpusrecallv1.Representation, error) {
	contractRepresentations := make(
		[]*corpusrecallv1.Representation, 0, len(recalledRepresentations),
	)
	for _, recalled := range recalledRepresentations {
		expressed, formed := forms.contractRepresentationFrom(recalled)
		if !formed {
			return nil, kindNotInContract(recalled.Kind)
		}
		contractRepresentations = append(contractRepresentations, expressed)
	}
	return contractRepresentations, nil
}

func (forms servedRepresentationForms) contractRepresentationFrom(
	recalled recall.RecalledRepresentation,
) (*corpusrecallv1.Representation, bool) {
	for _, form := range forms {
		if form.kind == recalled.Kind {
			return form.contractRepresentationFrom(recalled.Representation)
		}
	}
	return nil, false
}

func (forms servedRepresentationForms) contractRepresentationKindsFrom(
	kinds []recall.RepresentationKind,
) []corpusrecallv1.RepresentationKind {
	contractKinds := make([]corpusrecallv1.RepresentationKind, 0, len(kinds))
	for _, form := range forms {
		if slices.Contains(kinds, form.kind) {
			contractKinds = append(contractKinds, form.contractKind)
		}
	}
	return contractKinds
}
