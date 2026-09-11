package judgedqueries_test

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
)

type silentSeedlistObserver struct{}

func (silentSeedlistObserver) SeedlistRead(context.Context, string, int)          {}
func (silentSeedlistObserver) SeedlistUnreachable(context.Context, string, error) {}
func (silentSeedlistObserver) SeedlistUnreadable(context.Context, string, error)  {}

type silentPageReadingObserver struct{}

func (silentPageReadingObserver) PageReadingPerformed(
	context.Context,
	pagereading.PerformedPageReading,
) {
}
