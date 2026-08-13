package bolt

import (
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

func TestCapacityCauseOfClassifiesExhaustion(t *testing.T) {
	for name, testCase := range map[string]struct {
		err   error
		cause vault.WriteRefusalCause
	}{
		"no space":       {err: syscall.ENOSPC, cause: causeNoSpace},
		"over quota":     {err: syscall.EDQUOT, cause: causeOverQuota},
		"file too large": {err: syscall.EFBIG, cause: causeFileTooLarge},
	} {
		t.Run(name, func(t *testing.T) {
			cause, atCapacity := capacityCauseOf(fmt.Errorf("update storage: %w", testCase.err))
			if !atCapacity {
				t.Fatalf("capacityCauseOf(%v) reported room left", testCase.err)
			}
			if cause != testCase.cause {
				t.Errorf("cause = %q, want %q", cause, testCase.cause)
			}
		})
	}
}

func TestCapacityCauseOfPassesUnrelatedErrors(t *testing.T) {
	if _, atCapacity := capacityCauseOf(errors.New("checksum mismatch")); atCapacity {
		t.Error("capacityCauseOf reported an unrelated error as exhaustion")
	}
}

func TestCapacityErrorCarriesCauseAndSentinel(t *testing.T) {
	failure := capacityError{cause: causeNoSpace, err: vault.ErrAtCapacity}

	var carrier vault.WriteRefusal
	if !errors.As(error(failure), &carrier) {
		t.Fatal("capacityError does not satisfy vault.WriteRefusal")
	}
	if carrier.Cause() != causeNoSpace {
		t.Errorf("Cause() = %q, want %q", carrier.Cause(), causeNoSpace)
	}
	if !errors.Is(failure, vault.ErrAtCapacity) {
		t.Error("capacityError does not unwrap to vault.ErrAtCapacity")
	}
}
