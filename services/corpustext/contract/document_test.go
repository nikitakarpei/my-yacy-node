package searchdocument_test

import (
	_ "embed"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

//go:embed document.schema.json
var schema []byte

func TestDocumentFieldsMatchSchema(t *testing.T) {
	var parsed struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	schemaFields := make([]string, 0, len(parsed.Properties))
	for name := range parsed.Properties {
		schemaFields = append(schemaFields, name)
	}
	sort.Strings(schemaFields)

	sort.Strings(parsed.Required)
	if !reflect.DeepEqual(schemaFields, parsed.Required) {
		t.Fatalf("schema required %v does not cover properties %v", parsed.Required, schemaFields)
	}

	documentType := reflect.TypeOf(searchdocument.Document{})
	structFields := make([]string, 0, documentType.NumField())
	for i := range documentType.NumField() {
		structFields = append(structFields, documentType.Field(i).Tag.Get("json"))
	}
	sort.Strings(structFields)

	if !reflect.DeepEqual(structFields, schemaFields) {
		t.Errorf(
			"searchdocument.Document json tags %v do not match schema fields %v",
			structFields,
			schemaFields,
		)
	}
}
