// Package file reads release denials from a JSON file of the operator.
package file

import (
	"context"
	"fmt"
	"os"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

type Source struct {
	path string
}

func New(path string) Source {
	return Source{path: path}
}

func (s Source) Denials(_ context.Context) ([]trustedappreleases.Denial, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("read the release denials: %w", err)
	}

	return trustedappreleases.DenialsFrom(content)
}
