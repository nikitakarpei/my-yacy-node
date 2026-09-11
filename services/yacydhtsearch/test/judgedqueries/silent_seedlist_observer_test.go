package judgedqueries_test

import (
	"context"
)

type silentSeedlistObserver struct{}

func (silentSeedlistObserver) SeedlistRead(context.Context, string, int)          {}
func (silentSeedlistObserver) SeedlistUnreachable(context.Context, string, error) {}
func (silentSeedlistObserver) SeedlistUnreadable(context.Context, string, error)  {}
