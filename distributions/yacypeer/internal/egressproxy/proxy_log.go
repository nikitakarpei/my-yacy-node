package egressproxy

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"slices"

	"github.com/sirupsen/logrus"
)

const msgProxyEvent = "egress proxy event"

type proxyLog struct {
	ctx context.Context
}

func proxyLoggerFor(ctx context.Context) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrusLevelFor(ctx))
	logger.AddHook(proxyLog{ctx: ctx})
	return logger
}

func logrusLevelFor(ctx context.Context) logrus.Level {
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		return logrus.DebugLevel
	}
	return logrus.WarnLevel
}

func (proxyLog) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (l proxyLog) Fire(entry *logrus.Entry) error {
	attrs := make([]slog.Attr, 0, len(entry.Data)+1)
	attrs = append(attrs, slog.String("event", entry.Message))
	for _, key := range slices.Sorted(maps.Keys(entry.Data)) {
		attrs = append(attrs, slog.Any(key, entry.Data[key]))
	}
	slog.Default().LogAttrs(l.ctx, levelOf(entry.Level), msgProxyEvent, attrs...)
	return nil
}

func levelOf(level logrus.Level) slog.Level {
	switch level {
	case logrus.PanicLevel, logrus.FatalLevel:
		return slog.LevelError
	case logrus.ErrorLevel, logrus.WarnLevel:
		return slog.LevelWarn
	default:
		return slog.LevelDebug
	}
}
