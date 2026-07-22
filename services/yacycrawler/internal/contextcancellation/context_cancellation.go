// Package contextcancellation reports a cancelled context as a wrapped error.
package contextcancellation

import (
	"context"
	"fmt"
)

func Err(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}
	return nil
}
