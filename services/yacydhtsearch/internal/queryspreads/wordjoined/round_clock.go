package wordjoined

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type RoundTime struct {
	Budget    yacymodel.Optional[time.Duration]
	TimeSpent time.Duration
}

const (
	roundsLeftAtTheDiscovery   = 3
	roundsLeftAtTheCrossCheck  = 2
	roundsLeftAtTheURLMetadata = 1
)

type roundClock struct {
	budget    yacymodel.Optional[time.Duration]
	startedAt time.Time
}

func roundClockStartedWithin(ctx context.Context, roundsLeft int) roundClock {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return roundClock{budget: yacymodel.None[time.Duration](), startedAt: time.Now()}
	}

	return roundClock{
		budget:    yacymodel.Some(time.Until(deadline) / time.Duration(roundsLeft)),
		startedAt: time.Now(),
	}
}

func (clock roundClock) contextWithinTheBudget(
	ctx context.Context,
) (context.Context, context.CancelFunc) {
	budget, bounded := clock.budget.Get()
	if !bounded {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, budget)
}

func (clock roundClock) roundTimeOf(amountOfAsks int) yacymodel.Optional[RoundTime] {
	if amountOfAsks == 0 {
		return yacymodel.None[RoundTime]()
	}

	return yacymodel.Some(RoundTime{
		Budget:    clock.budget,
		TimeSpent: time.Since(clock.startedAt),
	})
}
