package grpc

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

var ErrRepresentationDoesNotMatchItsKind = errors.New(
	"representation does not match its kind",
)

func contractRepresentationsFrom(
	recalledRepresentations []recall.RecalledRepresentation,
) ([]*corpusrecallv1.Representation, error) {
	contractRepresentations := make(
		[]*corpusrecallv1.Representation, 0, len(recalledRepresentations),
	)
	for _, recalled := range recalledRepresentations {
		contractRepresentation, err := contractRepresentationFrom(recalled)
		if err != nil {
			return nil, err
		}
		contractRepresentations = append(contractRepresentations, contractRepresentation)
	}
	return contractRepresentations, nil
}

func contractRepresentationFrom(
	recalled recall.RecalledRepresentation,
) (*corpusrecallv1.Representation, error) {
	switch recalled.Kind {
	case markdown.Kind:
		page, isPage := recalled.Representation.(markdown.Page)
		if !isPage {
			return nil, representationNotMatchingItsKind(recalled)
		}
		return contractMarkdownRepresentationFrom(page), nil
	default:
		return nil, representationKindNotInTheContract(recalled.Kind)
	}
}

func contractMarkdownRepresentationFrom(page markdown.Page) *corpusrecallv1.Representation {
	return &corpusrecallv1.Representation{
		Representation: &corpusrecallv1.Representation_Markdown{
			Markdown: &corpusrecallv1.MarkdownRepresentation{
				CanonicalUrl: page.CanonicalURL,
				Markdown:     page.Markdown,
			},
		},
	}
}

func representationNotMatchingItsKind(recalled recall.RecalledRepresentation) error {
	return fmt.Errorf(
		"%w: %s holds %T",
		ErrRepresentationDoesNotMatchItsKind, recalled.Kind, recalled.Representation,
	)
}
