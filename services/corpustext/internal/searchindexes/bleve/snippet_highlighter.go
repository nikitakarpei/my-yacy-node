package bleve

import (
	"errors"

	blevesearch "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/highlight/format/plain"
	"github.com/blevesearch/bleve/v2/search/highlight/fragmenter/simple"
	simplehighlighter "github.com/blevesearch/bleve/v2/search/highlight/highlighter/simple"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
)

const (
	snippetFragmenter  = "corpus_text_snippet"
	snippetFormatter   = "corpus_text_snippet"
	snippetHighlighter = "corpus_text_snippet"
)

func snippetHighlighterDefined() error {
	cache := blevesearch.Config.Cache
	if _, err := cache.HighlighterNamed(snippetHighlighter); err == nil {
		return nil
	}
	_, fragmenterErr := cache.DefineFragmenter(snippetFragmenter, map[string]any{
		"type": simple.Name,
		"size": float64(textsearch.SnippetLength),
	})
	_, formatterErr := cache.DefineFragmentFormatter(snippetFormatter, map[string]any{
		"type":   plain.Name,
		"before": "",
		"after":  "",
	})
	_, highlighterErr := cache.DefineHighlighter(snippetHighlighter, map[string]any{
		"type":       simplehighlighter.Name,
		"fragmenter": snippetFragmenter,
		"formatter":  snippetFormatter,
		"separator":  "",
	})
	return errors.Join(fragmenterErr, formatterErr, highlighterErr)
}
