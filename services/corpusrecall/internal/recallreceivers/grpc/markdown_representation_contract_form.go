package grpc

import (
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

func markdownRepresentationContractForm() representationContractForm {
	return representationContractForm{
		representationKind:         markdown.Kind,
		contractRepresentationKind: corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
		contractRepresentationFrom: contractMarkdownRepresentationFrom,
	}
}

func contractMarkdownRepresentationFrom(
	representation recall.Representation,
) (*corpusrecallv1.Representation, bool) {
	page, isPage := representation.(markdown.Page)
	if !isPage {
		return nil, false
	}
	return &corpusrecallv1.Representation{
		Representation: &corpusrecallv1.Representation_Markdown{
			Markdown: &corpusrecallv1.MarkdownRepresentation{
				CanonicalUrl: page.CanonicalURL,
				Markdown:     page.Markdown,
			},
		},
	}, true
}
