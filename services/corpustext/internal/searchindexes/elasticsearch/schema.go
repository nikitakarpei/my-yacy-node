package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
)

const (
	textAnalyzerName       = "corpus_text"
	alreadyExistsException = "resource_already_exists_exception"
	detailLimit            = 4096
)

type Schema struct {
	endpoint string
	indexes  languageindex.LanguageIndexes
	client   *http.Client
}

func NewSchema(
	endpoint string,
	indexes languageindex.LanguageIndexes,
	client *http.Client,
) *Schema {
	return &Schema{
		endpoint: strings.TrimRight(endpoint, "/"),
		indexes:  indexes,
		client:   client,
	}
}

func (schema *Schema) Bootstrap(ctx context.Context) error {
	for _, index := range schema.indexes.All() {
		if err := schema.createIfAbsent(ctx, index); err != nil {
			return err
		}
		schema.reportMappingDrift(ctx, index)
	}
	return nil
}

func (schema *Schema) createIfAbsent(
	ctx context.Context,
	index languageindex.LanguageIndex,
) error {
	body, err := json.Marshal(definitionOf(index))
	if err != nil {
		return fmt.Errorf("marshal index definition %s: %w", index.Name, err)
	}
	status, detail, err := schema.send(ctx, http.MethodPut, index.Name, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create index %s: %w", index.Name, err)
	}
	if status < http.StatusMultipleChoices {
		slog.DebugContext(ctx, "elasticsearch index created", slog.String("index", index.Name))
		return nil
	}
	if bytes.Contains(detail, []byte(alreadyExistsException)) {
		slog.DebugContext(ctx, "elasticsearch index present", slog.String("index", index.Name))
		return nil
	}
	return fmt.Errorf("create index %s: status %d: %s", index.Name, status, detail)
}

func definitionOf(index languageindex.LanguageIndex) map[string]any {
	return map[string]any{
		"settings": map[string]any{
			"analysis": map[string]any{
				"analyzer": map[string]any{
					textAnalyzerName: analyzerOf(index.Language).definition(),
				},
			},
		},
		"mappings": map[string]any{"properties": documentProperties()},
	}
}

func documentProperties() map[string]any {
	return map[string]any{
		"title":      map[string]any{"type": "text", "analyzer": textAnalyzerName},
		"content":    map[string]any{"type": "text", "analyzer": textAnalyzerName},
		"url":        map[string]any{"type": "keyword"},
		"language":   map[string]any{"type": "keyword"},
		"crawled_at": map[string]any{"type": "date"},
	}
}

func (schema *Schema) send(
	ctx context.Context,
	method, path string,
	body io.Reader,
) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, schema.endpoint+"/"+path, body)
	if err != nil {
		return 0, nil, fmt.Errorf("build request %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := schema.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	detail, err := io.ReadAll(io.LimitReader(resp.Body, detailLimit))
	if err != nil {
		return 0, nil, fmt.Errorf("read response %s: %w", path, err)
	}
	return resp.StatusCode, detail, nil
}

func (schema *Schema) reportMappingDrift(
	ctx context.Context,
	index languageindex.LanguageIndex,
) {
	live, err := schema.livePropertiesOf(ctx, index.Name)
	if err != nil {
		slog.WarnContext(ctx, "elasticsearch mapping unreadable",
			slog.String("index", index.Name),
			slog.Any("error", err),
		)
		return
	}
	for field, expected := range documentProperties() {
		if reflect.DeepEqual(live[field], expected) {
			continue
		}
		slog.WarnContext(ctx, "elasticsearch mapping differs from the schema",
			slog.String("index", index.Name),
			slog.String("field", field),
			slog.Any("mapping", live[field]),
		)
	}
}

func (schema *Schema) livePropertiesOf(
	ctx context.Context,
	name string,
) (map[string]any, error) {
	status, detail, err := schema.send(ctx, http.MethodGet, name+"/_mapping", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("read mapping %s: status %d: %s", name, status, detail)
	}
	var mappings map[string]struct {
		Mappings struct {
			Properties map[string]any `json:"properties"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal(detail, &mappings); err != nil {
		return nil, fmt.Errorf("decode mapping %s: %w", name, err)
	}
	return mappings[name].Mappings.Properties, nil
}
