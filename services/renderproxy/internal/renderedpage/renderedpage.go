package renderedpage

import (
	"context"
	"errors"
)

var ErrTooLarge = errors.New("page exceeds the response byte limit")

type Page struct {
	StatusCode  int
	ContentType string
	Location    string
	Body        []byte
}

type Renderer interface {
	Render(ctx context.Context, targetURL string) (Page, error)
}
