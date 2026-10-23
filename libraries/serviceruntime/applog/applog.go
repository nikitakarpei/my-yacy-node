// Package applog installs the process-wide structured logger, emitting JSON to
// standard error at a level read from the LOG_LEVEL environment variable.
package applog

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const envLogLevel = "LOG_LEVEL"

func Configure(getenv func(string) string) error {
	handler, err := HandlerFor(getenv)
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(handler))
	return nil
}

func HandlerFor(getenv func(string) string) (slog.Handler, error) {
	level := slog.LevelInfo
	if raw := strings.TrimSpace(getenv(envLogLevel)); raw != "" {
		if err := level.UnmarshalText([]byte(strings.ToUpper(raw))); err != nil {
			return nil, fmt.Errorf("%s: %w", envLogLevel, err)
		}
	}
	return slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}), nil
}
