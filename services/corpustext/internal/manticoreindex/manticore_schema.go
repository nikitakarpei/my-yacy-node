package manticoreindex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
)

const detailLimit = 4096

type ManticoreSchema struct {
	endpoint string
	indexes  languageindex.LanguageIndexes
	client   *http.Client
}

func NewManticoreSchema(
	endpoint string,
	indexes languageindex.LanguageIndexes,
	client *http.Client,
) *ManticoreSchema {
	return &ManticoreSchema{
		endpoint: strings.TrimRight(endpoint, "/"),
		indexes:  indexes,
		client:   client,
	}
}

func (schema *ManticoreSchema) Bootstrap(ctx context.Context) error {
	for _, index := range schema.indexes.All() {
		if err := schema.createIfAbsent(ctx, index); err != nil {
			return err
		}
		schema.reportColumnDrift(ctx, index)
	}
	return schema.recreateFanOutTable(ctx)
}

func (schema *ManticoreSchema) createIfAbsent(
	ctx context.Context,
	index languageindex.LanguageIndex,
) error {
	statement := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (%s)",
		index.Name,
		strings.Join(columnDeclarations(), ", "),
	)
	if morphology := morphologyOf(index.Language); morphology != "" {
		statement += fmt.Sprintf(" morphology='%s'", morphology)
	}
	if _, err := schema.rowsOf(ctx, statement); err != nil {
		return fmt.Errorf("create table %s: %w", index.Name, err)
	}
	slog.DebugContext(ctx, "manticore table present", slog.String("table", index.Name))
	return nil
}

type tableColumn struct {
	name     string
	dataType string
}

var documentColumns = []tableColumn{
	{name: "title", dataType: "text"},
	{name: "content", dataType: "text"},
	{name: "url", dataType: "string"},
	{name: "language", dataType: "string"},
	{name: "crawled_at", dataType: "string"},
}

func columnDeclarations() []string {
	declarations := make([]string, 0, len(documentColumns))
	for _, column := range documentColumns {
		declarations = append(declarations, column.name+" "+column.dataType)
	}
	return declarations
}

func (schema *ManticoreSchema) rowsOf(
	ctx context.Context,
	statement string,
) ([]map[string]any, error) {
	form := url.Values{"query": {statement}}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		schema.endpoint+"/sql?mode=raw",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("build statement request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := schema.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("run statement: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	detail, err := io.ReadAll(io.LimitReader(resp.Body, detailLimit))
	if err != nil {
		return nil, fmt.Errorf("read statement response: %w", err)
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("run statement: status %d: %s", resp.StatusCode, detail)
	}
	return rowsIn(detail)
}

type statementOutcome struct {
	Data  []map[string]any `json:"data"`
	Error string           `json:"error"`
}

func rowsIn(detail []byte) ([]map[string]any, error) {
	var outcomes []statementOutcome
	if err := json.Unmarshal(detail, &outcomes); err != nil {
		return nil, fmt.Errorf("decode statement response: %w", err)
	}
	rows := make([]map[string]any, 0, len(outcomes))
	for _, outcome := range outcomes {
		if outcome.Error != "" {
			return nil, errors.New(outcome.Error)
		}
		rows = append(rows, outcome.Data...)
	}
	return rows, nil
}

func (schema *ManticoreSchema) reportColumnDrift(
	ctx context.Context,
	index languageindex.LanguageIndex,
) {
	rows, err := schema.rowsOf(ctx, "DESC "+index.Name)
	if err != nil {
		slog.WarnContext(ctx, "manticore table schema unreadable",
			slog.String("table", index.Name),
			slog.Any("error", err),
		)
		return
	}
	live := columnTypesIn(rows)
	for _, column := range documentColumns {
		if live[column.name] == column.dataType {
			continue
		}
		slog.WarnContext(ctx, "manticore table differs from the schema",
			slog.String("table", index.Name),
			slog.String("column", column.name),
			slog.String("type", live[column.name]),
		)
	}
}

func columnTypesIn(rows []map[string]any) map[string]string {
	types := make(map[string]string, len(rows))
	for _, row := range rows {
		name, _ := row["Field"].(string)
		dataType, _ := row["Type"].(string)
		types[name] = dataType
	}
	return types
}

func (schema *ManticoreSchema) recreateFanOutTable(ctx context.Context) error {
	prefix := schema.indexes.Prefix()
	if _, err := schema.rowsOf(ctx, "DROP TABLE IF EXISTS "+prefix); err != nil {
		return fmt.Errorf("drop table %s: %w", prefix, err)
	}
	statement := "CREATE TABLE " + prefix + " type='distributed'"
	for _, index := range schema.indexes.All() {
		statement += fmt.Sprintf(" local='%s'", index.Name)
	}
	if _, err := schema.rowsOf(ctx, statement); err != nil {
		return fmt.Errorf("create table %s: %w", prefix, err)
	}
	slog.DebugContext(ctx, "manticore fan-out table created", slog.String("table", prefix))
	return nil
}
