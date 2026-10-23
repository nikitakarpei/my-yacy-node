package main

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

type searchIndexSchema interface {
	Bootstrap(ctx context.Context) error
}

type searchIndexSelection struct {
	index   pageintake.SearchIndex
	schema  searchIndexSchema
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
		languageIndexes, err := languageindex.IndexesFor(DefaultIndexBaseName, cfg.Languages)
		if err != nil {
			return searchIndexSelection{}, err
		}
		indexes := bleve.NewIndexes(cfg.BleveDirectory, languageIndexes)
		return searchIndexSelection{
			index:   indexes,
			schema:  indexes,
			matcher: indexes,
			prefix:  languageIndexes.Prefix(),
			close:   indexes.Close,
		}, nil
	case SearchIndexEngineElasticsearch:
		indexes, err := languageindex.IndexesFor(cfg.ElasticsearchIndex, cfg.Languages)
		if err != nil {
			return searchIndexSelection{}, err
		}
		return searchIndexSelection{
			index: elasticsearch.New(
				cfg.ElasticsearchURL, indexes, client,
			),
			schema: elasticsearch.NewSchema(
				cfg.ElasticsearchURL, indexes, client,
			),
			matcher: elasticsearch.NewMatcher(
				cfg.ElasticsearchURL, indexes, client,
			),
			prefix: indexes.Prefix(),
			close:  closeNothing,
		}, nil
	case SearchIndexEngineManticore:
		tables, err := languageindex.IndexesFor(cfg.ManticoreTable, cfg.Languages)
		if err != nil {
			return searchIndexSelection{}, err
		}
		return searchIndexSelection{
			index:   manticore.New(cfg.ManticoreURL, tables, client),
			schema:  manticore.NewSchema(cfg.ManticoreURL, tables, client),
			matcher: manticore.NewMatcher(cfg.ManticoreURL, tables, client),
			prefix:  tables.Prefix(),
			close:   closeNothing,
		}, nil
	default:
		return searchIndexSelection{}, unknownSearchIndexEngine(cfg.SearchIndexEngine)
	}
}
