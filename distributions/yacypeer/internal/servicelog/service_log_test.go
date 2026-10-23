package servicelog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/servicelog"
)

func loggerInto(buffer *bytes.Buffer) *slog.Logger {
	return slog.New(servicelog.NewHandler(slog.NewJSONHandler(buffer, nil)))
}

func lineFrom(t *testing.T, buffer *bytes.Buffer) map[string]any {
	t.Helper()
	var line map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &line); err != nil {
		t.Fatalf("decode the log line %q: %v", buffer, err)
	}
	return line
}

func TestHandleNamesTheServiceOfTheContext(t *testing.T) {
	var buffer bytes.Buffer
	ctx := servicelog.ContextFor(context.Background(), "corpustext")

	loggerInto(&buffer).With(slog.String("engine", "bleve")).InfoContext(ctx, "started")

	line := lineFrom(t, &buffer)
	if line["service"] != "corpustext" || line["engine"] != "bleve" {
		t.Errorf("log line = %v, want service corpustext and engine bleve", line)
	}
}

func TestHandleLeavesALineWithoutAServiceUnnamed(t *testing.T) {
	var buffer bytes.Buffer

	loggerInto(&buffer).WithGroup("storage").InfoContext(context.Background(), "closed")

	if line := lineFrom(t, &buffer); line["service"] != nil {
		t.Errorf("log line = %v, want no service", line)
	}
}

func TestEnabledFollowsTheNextHandler(t *testing.T) {
	handler := servicelog.NewHandler(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	if handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("info enabled below a warn handler")
	}
}
