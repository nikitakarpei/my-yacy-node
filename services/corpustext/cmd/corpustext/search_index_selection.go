package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/elasticsearch"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/manticore"
)

type searchIndexSchema interface {
	Bootstrap(ctx context.Context) error
}

type searchIndexSelection struct {
	index  pageintake.SearchIndex
	schema searchIndexSchema
	prefix string
}

func selectSearchIndex(
	cfg ServiceConfig,
	client *http.Client,
) (searchIndexSelection, error) {
	switch cfg.SearchIndexEngine {
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
			prefix: indexes.Prefix(),
		}, nil
	case SearchIndexEngineManticore:
		tables, err := languageindex.IndexesFor(cfg.ManticoreTable, cfg.Languages)
		if err != nil {
			return searchIndexSelection{}, err
		}
		return searchIndexSelection{
			index:  manticore.New(cfg.ManticoreURL, tables, client),
			schema: manticore.NewSchema(cfg.ManticoreURL, tables, client),
			prefix: tables.Prefix(),
		}, nil
	default:
		return searchIndexSelection{}, fmt.Errorf(
			"%s: unknown engine %q", EnvSearchIndexEngine, cfg.SearchIndexEngine,
		)
	}
}
