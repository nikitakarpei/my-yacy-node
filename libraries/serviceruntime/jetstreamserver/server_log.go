package jetstreamserver

import (
	"context"
	"fmt"
	"log/slog"
)

type serverLog struct{}

func (serverLog) Noticef(format string, v ...any) {
	slog.DebugContext(context.Background(), "nats server notice", reportOf(format, v))
}

func (serverLog) Warnf(format string, v ...any) {
	slog.WarnContext(context.Background(), "nats server warning", reportOf(format, v))
}

func (serverLog) Fatalf(format string, v ...any) {
	slog.ErrorContext(context.Background(), "nats server failed", reportOf(format, v))
}

func (serverLog) Errorf(format string, v ...any) {
	slog.ErrorContext(context.Background(), "nats server error", reportOf(format, v))
}

func (serverLog) Debugf(format string, v ...any) {
	slog.DebugContext(context.Background(), "nats server debug", reportOf(format, v))
}

func (serverLog) Tracef(format string, v ...any) {
	slog.DebugContext(context.Background(), "nats server trace", reportOf(format, v))
}

func reportOf(format string, v []any) slog.Attr {
	return slog.String("report", fmt.Sprintf(format, v...))
}
