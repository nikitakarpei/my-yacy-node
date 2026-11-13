// Package http loads the Android trust: the revocation list of Android attestation keys and the
// GrapheneOS boot key list, from the addresses where Google and GrapheneOS publish them.
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

func (s Source) Latest(ctx context.Context) (*android.Trust, error) {
	androidTrust, err := android.LoadTrust(ctx, s.client)
	if err != nil {
		return nil, fmt.Errorf("load the Android trust: %w", err)
	}

	return androidTrust, nil
}
