// Package tuf takes the Sigstore trusted root from the Sigstore TUF repository and keeps its
// metadata in a cache directory. When the repository cannot be reached, a cache whose metadata
// has not expired gives the root.
package tuf

import (
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/root"
	sigstoretuf "github.com/sigstore/sigstore-go/pkg/tuf"
)

type TrustedRoots struct {
	cachePath string
}

func New(cachePath string) TrustedRoots {
	return TrustedRoots{cachePath: cachePath}
}

func (r TrustedRoots) TrustedRoot() (root.TrustedMaterial, error) {
	trustedRoot, err := root.FetchTrustedRootWithOptions(
		sigstoretuf.DefaultOptions().WithCachePath(r.cachePath),
	)
	if err != nil {
		return nil, fmt.Errorf("fetch the Sigstore trusted root: %w", err)
	}

	return trustedRoot, nil
}
