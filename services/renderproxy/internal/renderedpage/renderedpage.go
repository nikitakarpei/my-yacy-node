// Package renderedpage holds the page a proxied GET is answered with.
package renderedpage

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagefreshness"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagereplay"
)

var ErrTooLarge = errors.New("page exceeds the response byte limit")

type Target struct {
	URL        string
	Conditions pagefreshness.Conditions
}

type Page struct {
	StatusCode   int
	ContentType  string
	Location     string
	ReuseTerms   pagefreshness.ReuseTerms
	CaptureTerms pagereplay.CaptureTerms
	Body         []byte
}

type Renderer interface {
	Render(ctx context.Context, target Target) (Page, error)
}
