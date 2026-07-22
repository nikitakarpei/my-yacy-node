package pageabsorption

import "context"

type RedirectResolver interface {
	Record(ctx context.Context, requested, canonical string) error
}
