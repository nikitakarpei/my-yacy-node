package corpustextassembly

import (
	"context"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/bleve"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/elasticsearch"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/manticore"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
)

type searchIndexSelection struct {
	index   pageintake.SearchIndex
	open    func(ctx context.Context) error
	matcher textsearch.Matcher
	prefix  string
	close   func(ctx context.Context)
}

func closeNothing(context.Context) {}

func selectSearchIndex(
	cfg ServiceConfig,
	client *http.Client,
) (searchIndexSelection, error) {
	switch cfg.SearchIndexEngine {
	case SearchIndexEngineBleve:
		languageIndexes, err := languageindex.IndexesFor(
			DefaultIndexBaseName, bleve.SchemaVersion, cfg.Languages,
		)
		if err != nil {
			return searchIndexSelection{}, err
		}
		bleveIndexes := bleve.New(cfg.BleveDirectory, languageIndexes)
		return searchIndexSelection{
			index:   bleveIndexes,
			open:    bleveIndexes.Open,
			matcher: bleveIndexes,
			prefix:  languageIndexes.Prefix(),
			close:   bleveIndexes.Close,
		}, nil
	case SearchIndexEngineElasticsearch:
		indexes, err := languageindex.IndexesFor(
			cfg.ElasticsearchIndex, elasticsearch.SchemaVersion, cfg.Languages,
		)
		if err != nil {
			return searchIndexSelection{}, err
		}
		return searchIndexSelection{
			index: elasticsearch.New(
				cfg.ElasticsearchURL, indexes, client,
			),
			open: elasticsearch.NewSchema(
				cfg.ElasticsearchURL, indexes, client,
			).Bootstrap,
			matcher: elasticsearch.NewMatcher(
				cfg.ElasticsearchURL, indexes, client,
			),
			prefix: indexes.Prefix(),
			close:  closeNothing,
		}, nil
	case SearchIndexEngineManticore:
		tables, err := languageindex.IndexesFor(
			cfg.ManticoreTable, manticore.SchemaVersion, cfg.Languages,
		)
		if err != nil {
			return searchIndexSelection{}, err
		}
		return searchIndexSelection{
			index:   manticore.New(cfg.ManticoreURL, tables, client),
			open:    manticore.NewSchema(cfg.ManticoreURL, tables, client).Bootstrap,
			matcher: manticore.NewMatcher(cfg.ManticoreURL, tables, client),
			prefix:  tables.Prefix(),
			close:   closeNothing,
		}, nil
	default:
		return searchIndexSelection{}, unknownSearchIndexEngine(cfg.SearchIndexEngine)
	}
}
