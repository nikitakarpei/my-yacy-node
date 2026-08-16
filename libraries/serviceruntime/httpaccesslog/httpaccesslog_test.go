package httpaccesslog_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
)

type recordedLines struct {
	slog.Handler
	levels []slog.Level
}

func (r *recordedLines) Enabled(context.Context, slog.Level) bool { return true }

func (r *recordedLines) Handle(_ context.Context, record slog.Record) error {
	r.levels = append(r.levels, record.Level)

	return nil
}

func observeStatus(t *testing.T, status int) slog.Level {
	t.Helper()

	lines := &recordedLines{Handler: slog.Default().Handler()}
	previous := slog.Default()
	slog.SetDefault(slog.New(lines))
	t.Cleanup(func() { slog.SetDefault(previous) })

	httpaccesslog.New().ObserveRequest(
		context.Background(),
		httpobservation.ServedRequest{Status: status},
	)

	if len(lines.levels) != 1 {
		t.Fatalf("logged %d lines, want 1", len(lines.levels))
	}

	return lines.levels[0]
}

func TestAServedRequestIsLoggedAtDebug(t *testing.T) {
	if level := observeStatus(t, http.StatusOK); level != slog.LevelDebug {
		t.Errorf("level = %v, want DEBUG", level)
	}
}

func TestAFailedRequestIsLoggedAtWarn(t *testing.T) {
	if level := observeStatus(t, http.StatusBadRequest); level != slog.LevelWarn {
		t.Errorf("level = %v, want WARN", level)
	}
}
