package contentextraction

import (
	"context"
	"errors"
)

var ErrNestingTooDeep = errors.New("container nesting too deep")

type ContainerExpander interface {
	Expand(
		ctx context.Context,
		containerURL, contentType string,
		body []byte,
	) ([]ContainerMember, error)
}
