// Package memberlog names the member of the peer in each log line the member writes with
// its run context.
package memberlog

import (
	"context"
	"fmt"
	"log/slog"
)

const memberAttribute = "member"

type memberKey struct{}

func ContextFor(ctx context.Context, member string) context.Context {
	return context.WithValue(ctx, memberKey{}, member)
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
	if member, named := ctx.Value(memberKey{}).(string); named {
		record = record.Clone()
		record.AddAttrs(slog.String(memberAttribute, member))
	}
	if err := h.next.Handle(ctx, record); err != nil {
		return fmt.Errorf("member log: %w", err)
	}
	return nil
}

func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return Handler{next: h.next.WithAttrs(attrs)}
}

func (h Handler) WithGroup(name string) slog.Handler {
	return Handler{next: h.next.WithGroup(name)}
}
