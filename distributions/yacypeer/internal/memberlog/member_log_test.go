package memberlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/memberlog"
)

func loggerInto(buffer *bytes.Buffer) *slog.Logger {
	return slog.New(memberlog.NewHandler(slog.NewJSONHandler(buffer, nil)))
}

func lineFrom(t *testing.T, buffer *bytes.Buffer) map[string]any {
	t.Helper()
	var line map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &line); err != nil {
		t.Fatalf("decode the log line %q: %v", buffer, err)
	}
	return line
}

func TestHandleNamesTheMemberOfTheContext(t *testing.T) {
	var buffer bytes.Buffer
	ctx := memberlog.ContextFor(context.Background(), "corpustext")

	loggerInto(&buffer).With(slog.String("engine", "bleve")).InfoContext(ctx, "started")

	line := lineFrom(t, &buffer)
	if line["member"] != "corpustext" || line["engine"] != "bleve" {
		t.Errorf("log line = %v, want member corpustext and engine bleve", line)
	}
}

func TestHandleLeavesALineWithoutAMemberUnnamed(t *testing.T) {
	var buffer bytes.Buffer

	loggerInto(&buffer).WithGroup("storage").InfoContext(context.Background(), "closed")

	if line := lineFrom(t, &buffer); line["member"] != nil {
		t.Errorf("log line = %v, want no member", line)
	}
}

func TestEnabledFollowsTheNextHandler(t *testing.T) {
	handler := memberlog.NewHandler(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	if handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("info enabled below a warn handler")
	}
}
