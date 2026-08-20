package boltvault_test

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestAWriteThatExhaustsTheFilesystemReportsItsCause(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		exhaustion error
		cause      vault.WriteRefusalCause
	}{
		{name: "NoSpace", exhaustion: syscall.ENOSPC, cause: "no_space"},
		{name: "OverQuota", exhaustion: syscall.EDQUOT, cause: "over_quota"},
		{name: "FileTooLarge", exhaustion: syscall.EFBIG, cause: "file_too_large"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			engine := openEngine(t)

			err := engine.Update(
				context.Background(),
				func(vault.EngineTxn) error {
					return fmt.Errorf("write page: %w", testCase.exhaustion)
				},
			)
			if !errors.Is(err, vault.ErrAtCapacity) {
				t.Fatalf("Update = %v, want %v", err, vault.ErrAtCapacity)
			}

			var refusal vault.WriteRefusal
			if !errors.As(err, &refusal) {
				t.Fatalf("Update error %v carries no refusal cause", err)
			}
			if refusal.Cause() != testCase.cause {
				t.Fatalf("Cause = %q, want %q", refusal.Cause(), testCase.cause)
			}
		})
	}
}
