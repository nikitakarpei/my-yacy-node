package crawlcapability

import (
	"context"
	"errors"
)

var ErrContainerOverflow = errors.New("container overflow")

type ArchiveExpansion interface {
	Expand(
		ctx context.Context,
		containerURL, contentType string,
		body []byte,
	) ([]ArchiveMember, error)
}
