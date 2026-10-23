package bleve

import (
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/blevesearch/bleve/v2/search/highlight"
	"github.com/blevesearch/bleve/v2/search/highlight/format/plain"
	"github.com/blevesearch/bleve/v2/search/highlight/fragmenter/simple"
	simplehighlighter "github.com/blevesearch/bleve/v2/search/highlight/highlighter/simple"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
)

const snippetHighlighter = "corpus_text_snippet"

func init() {
	if err := registry.RegisterHighlighter(snippetHighlighter, snippetHighlighterFrom); err != nil {
		panic(err)
	}
}

func snippetHighlighterFrom(
	_ map[string]any,
	_ *registry.Cache,
) (highlight.Highlighter, error) {
	return simplehighlighter.NewHighlighter(
		simple.NewFragmenter(textsearch.SnippetLength),
		plain.NewFragmentFormatter("", ""),
		"",
	), nil
}
