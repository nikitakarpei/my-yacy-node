package contentextraction

import (
	"context"
	"errors"
)

var ErrContainerOverflow = errors.New("container overflow")

type Container interface {
	Expand(
		ctx context.Context,
		containerURL, contentType string,
		body []byte,
	) ([]ArchiveMember, error)
}
