// Package servicelog names the service of the peer in each log line the service writes with
// its run context.
package servicelog

import (
	"context"
	"fmt"
	"log/slog"
)

const serviceAttribute = "service"

type serviceKey struct{}

func ContextFor(ctx context.Context, service string) context.Context {
	return context.WithValue(ctx, serviceKey{}, service)
}

type Handler struct {
	next slog.Handler
}

func NewHandler(next slog.Handler) Handler {
	return Handler{next: next}
}

func (h Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h Handler) Handle(ctx context.Context, record slog.Record) error {
	if service, named := ctx.Value(serviceKey{}).(string); named {
		record = record.Clone()
		record.AddAttrs(slog.String(serviceAttribute, service))
	}
	if err := h.next.Handle(ctx, record); err != nil {
		return fmt.Errorf("service log: %w", err)
	}
	return nil
}

func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return Handler{next: h.next.WithAttrs(attrs)}
}

func (h Handler) WithGroup(name string) slog.Handler {
	return Handler{next: h.next.WithGroup(name)}
}
