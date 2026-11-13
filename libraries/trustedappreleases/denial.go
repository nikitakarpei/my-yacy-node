package trustedappreleases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMalformedDenials = errors.New("the release denials are malformed")
	earliestDeniedFrom  = time.Unix(1, 0)
)

type DenialSource interface {
	Denials(ctx context.Context) ([]Denial, error)
}

type Denial struct {
	DeniedFrom         time.Time `json:"deniedFrom"`
	SigningCertificate string    `json:"signingCertificate"`
}

type releaseDenials struct {
	Schema  string   `json:"$schema"`
	Denials []Denial `json:"denials"`
}

func DenialsFrom(content []byte) ([]Denial, error) {
	var document releaseDenials
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedDenials, err)
	}
	if document.Denials == nil {
		return nil, fmt.Errorf("%w: no denials list", ErrMalformedDenials)
	}
	denials := document.Denials
	for i, denial := range denials {
		denials[i].SigningCertificate = signingCertificateFrom(denial.SigningCertificate)
		if !isSigningCertificate(denials[i].SigningCertificate) ||
			denial.DeniedFrom.Before(earliestDeniedFrom) {
			return nil, fmt.Errorf("%w: %+v", ErrMalformedDenials, denial)
		}
	}

	return denials, nil
}

func signingCertificateFrom(writtenDigest string) string {
	return strings.ToLower(strings.ReplaceAll(writtenDigest, ":", ""))
}
