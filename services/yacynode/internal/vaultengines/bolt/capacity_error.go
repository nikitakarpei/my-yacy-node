package bolt

import (
	"errors"
	"syscall"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	causeNoSpace      vault.WriteRefusalCause = "no_space"
	causeOverQuota    vault.WriteRefusalCause = "over_quota"
	causeFileTooLarge vault.WriteRefusalCause = "file_too_large"
)

type capacityError struct {
	cause vault.WriteRefusalCause
	err   error
}

func (e capacityError) Error() string { return e.err.Error() }

func (e capacityError) Unwrap() error { return e.err }

func (e capacityError) Cause() vault.WriteRefusalCause { return e.cause }

// TECHDEBT: testing — the exhaustion causes are untested; no exported seam makes bolt run out
func capacityCauseOf(err error) (vault.WriteRefusalCause, bool) {
	switch {
	case errors.Is(err, syscall.ENOSPC):
		return causeNoSpace, true
	case errors.Is(err, syscall.EDQUOT):
		return causeOverQuota, true
	case errors.Is(err, syscall.EFBIG):
		return causeFileTooLarge, true
	}

	return vault.WriteRefusalCause(""), false
}
