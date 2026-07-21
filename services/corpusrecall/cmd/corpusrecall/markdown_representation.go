package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/markdownrecall"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerecall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

const markdownKind pagerecall.Kind = "markdown"

func markdownRepresentation(source pagerecall.Source) representationKind {
	return representationKind{
		kind:      markdownKind,
		protoKind: corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
		source:    source,
		encode:    encodeMarkdown,
	}
}

func encodeMarkdown(representation pagerecall.Representation) *corpusrecallv1.Representation {
	page := representation.(markdownrecall.MarkdownPage)
	return &corpusrecallv1.Representation{
		Representation: &corpusrecallv1.Representation_Markdown{
			Markdown: &corpusrecallv1.MarkdownRepresentation{
				CanonicalUrl: page.CanonicalURL,
				Markdown:     page.Markdown,
			},
		},
	}
}
