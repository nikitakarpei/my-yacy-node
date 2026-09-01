package vaultenginetest

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var errDiscardedAttempt = errors.New("discarded repeat attempt")

// EngineRepeatingWrites runs each write closure two times and rolls the first
// run back, so a closure that leaves an effect outside its transaction reports it.
func EngineRepeatingWrites(engine vault.Engine) vault.Engine {
	return repeatingEngine{Engine: engine}
}

type repeatingEngine struct {
	vault.Engine
}

func (e repeatingEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if err := e.discardedAttempt(ctx, fn); err != nil {
		return err
	}

	return e.Engine.Update(ctx, fn)
}

func (e repeatingEngine) discardedAttempt(
	ctx context.Context,
	fn func(vault.EngineTxn) error,
) error {
	err := e.Engine.Update(ctx, func(etx vault.EngineTxn) error {
		if closureErr := fn(etx); closureErr != nil {
			return closureErr
		}

		return errDiscardedAttempt
	})
	if errors.Is(err, errDiscardedAttempt) {
		return nil
	}

	return err
}
