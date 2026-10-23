package applog_test

import (
	"log/slog"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
)

func fixedEnv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestConfigure(t *testing.T) {
	if err := applog.Configure(fixedEnv(map[string]string{"LOG_LEVEL": "debug"})); err != nil {
		t.Fatalf("valid level: %v", err)
	}
	if err := applog.Configure(fixedEnv(map[string]string{})); err != nil {
		t.Fatalf("default level: %v", err)
	}
	if err := applog.Configure(fixedEnv(map[string]string{"LOG_LEVEL": "nonsense"})); err == nil {
		t.Fatal("invalid level should error")
	}
}

func TestHandlerForFiltersBelowTheLevel(t *testing.T) {
	handler, err := applog.HandlerFor(fixedEnv(map[string]string{"LOG_LEVEL": "warn"}))
	if err != nil {
		t.Fatalf("HandlerFor: %v", err)
	}
	if handler.Enabled(t.Context(), slog.LevelInfo) {
		t.Error("info enabled at level warn")
	}
}
