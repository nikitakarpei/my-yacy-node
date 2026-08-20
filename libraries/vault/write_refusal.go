package vault

import "errors"

const unclassifiedCause = "unclassified"

type WriteRefusalCause string

// WriteRefusal is an error the storage engine attaches a Cause to. The engine
// implementing it declares a bounded set of causes, since the value becomes a
// metric label.
type WriteRefusal interface {
	error
	Cause() WriteRefusalCause
}

func refusalCauseOf(err error) WriteRefusalCause {
	var refusal WriteRefusal
	if errors.As(err, &refusal) {
		return refusal.Cause()
	}

	return unclassifiedCause
}
