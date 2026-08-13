package manticore_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/manticore"
)

func tablesFor(t *testing.T) languageindex.LanguageIndexes {
	t.Helper()
	tables, err := languageindex.IndexesFor("yacy_text", []string{"en"})
	if err != nil {
		t.Fatalf("tables: %v", err)
	}
	return tables
}

func statementRecorder(t *testing.T, statements *[]string, response string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		*statements = append(*statements, r.PostForm.Get("query"))
		_, _ = io.WriteString(w, response)
	}))
}

func TestManticoreSchemaStemsOnlyTheLanguageTable(t *testing.T) {
	var statements []string
	server := statementRecorder(t, &statements, `[{"total":0,"error":"","warning":""}]`)
	defer server.Close()

	schema := manticore.NewSchema(server.URL, tablesFor(t), server.Client())
	if err := schema.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	english := creationOf(t, statements, "yacy_text_v1_en")
	if !strings.Contains(english, "morphology='stem_en'") {
		t.Errorf("english table = %q, want it stemmed", english)
	}
	catchAll := creationOf(t, statements, "yacy_text_v1_und")
	if strings.Contains(catchAll, "morphology") {
		t.Errorf("catch-all table = %q, want no morphology", catchAll)
	}
}

func TestManticoreSchemaRecreatesTheFanOutTableOverEveryLanguage(t *testing.T) {
	var statements []string
	server := statementRecorder(t, &statements, `[{"total":0,"error":"","warning":""}]`)
	defer server.Close()

	schema := manticore.NewSchema(server.URL, tablesFor(t), server.Client())
	if err := schema.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	if !contains(statements, "DROP TABLE IF EXISTS yacy_text_v1") {
		t.Errorf("statements = %v, want the fan-out table dropped", statements)
	}
	fanOut := creationOf(t, statements, "yacy_text_v1")
	for _, table := range []string{"yacy_text_v1_und", "yacy_text_v1_en"} {
		if !strings.Contains(fanOut, "local='"+table+"'") {
			t.Errorf("fan-out table = %q, want it to span %q", fanOut, table)
		}
	}
}

func creationOf(t *testing.T, statements []string, table string) string {
	t.Helper()
	for _, statement := range statements {
		if strings.HasPrefix(statement, "CREATE TABLE") &&
			strings.Contains(statement, table+" ") {
			return statement
		}
	}
	t.Fatalf("statements = %v, want table %q created", statements, table)
	return ""
}

func contains(statements []string, wanted string) bool {
	for _, statement := range statements {
		if statement == wanted {
			return true
		}
	}
	return false
}

func TestManticoreSchemaFailsOnStatementError(t *testing.T) {
	var statements []string
	server := statementRecorder(t, &statements, `[{"total":0,"error":"syntax error","warning":""}]`)
	defer server.Close()

	schema := manticore.NewSchema(server.URL, tablesFor(t), server.Client())
	if err := schema.Bootstrap(context.Background()); err == nil {
		t.Fatal("expected error when the statement is rejected")
	}
}
