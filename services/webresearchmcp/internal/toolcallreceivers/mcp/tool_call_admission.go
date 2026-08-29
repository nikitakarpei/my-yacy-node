package mcp

import (
	"context"
	"fmt"
)

type toolCallAdmission struct {
	places chan struct{}
}

func newToolCallAdmission(toolCallConcurrency int) *toolCallAdmission {
	return &toolCallAdmission{places: make(chan struct{}, toolCallConcurrency)}
}

func (a *toolCallAdmission) admit(ctx context.Context) error {
	select {
	case a.places <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for a place among the tool calls under way: %w", ctx.Err())
	}
}

func (a *toolCallAdmission) release() {
	<-a.places
}
