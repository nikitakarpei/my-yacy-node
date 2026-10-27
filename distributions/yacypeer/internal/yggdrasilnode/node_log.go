package yggdrasilnode

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

const msgNodeEvent = "yggdrasil event"

type nodeLog struct {
	ctx context.Context
}

func nodeLogFor(ctx context.Context) nodeLog {
	return nodeLog{ctx: ctx}
}

func (l nodeLog) Printf(format string, values ...any) { l.debug(fmt.Sprintf(format, values...)) }
func (l nodeLog) Println(values ...any)               { l.debug(fmt.Sprintln(values...)) }
func (l nodeLog) Infof(format string, values ...any)  { l.debug(fmt.Sprintf(format, values...)) }
func (l nodeLog) Infoln(values ...any)                { l.debug(fmt.Sprintln(values...)) }
func (l nodeLog) Debugf(format string, values ...any) { l.debug(fmt.Sprintf(format, values...)) }
func (l nodeLog) Debugln(values ...any)               { l.debug(fmt.Sprintln(values...)) }
func (l nodeLog) Traceln(values ...any)               { l.debug(fmt.Sprintln(values...)) }

func (l nodeLog) Warnf(format string, values ...any) {
	l.log(slog.LevelWarn, fmt.Sprintf(format, values...))
}

func (l nodeLog) Warnln(values ...any) {
	l.log(slog.LevelWarn, fmt.Sprintln(values...))
}

func (l nodeLog) Errorf(format string, values ...any) {
	l.log(slog.LevelError, fmt.Sprintf(format, values...))
}

func (l nodeLog) Errorln(values ...any) {
	l.log(slog.LevelError, fmt.Sprintln(values...))
}

func (l nodeLog) debug(event string) {
	l.log(slog.LevelDebug, event)
}

func (l nodeLog) log(level slog.Level, event string) {
	slog.Default().LogAttrs(l.ctx, level, msgNodeEvent,
		slog.String("event", strings.TrimSpace(event)),
	)
}
