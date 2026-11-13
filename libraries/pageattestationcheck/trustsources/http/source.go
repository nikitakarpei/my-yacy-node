// Package http loads the revocation list of Android attestation keys and the GrapheneOS boot key
// list from the addresses where Google and GrapheneOS publish them.
package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"
)

type Source struct {
	client *http.Client
}

func New(client *http.Client) Source {
	return Source{client: client}
}

func (s Source) Trust(ctx context.Context) (*android.Trust, error) {
	trust, err := android.LoadTrust(ctx, s.client)
	if err != nil {
		return nil, fmt.Errorf("load the revocation list of attestation keys: %w", err)
	}

	return trust, nil
}
