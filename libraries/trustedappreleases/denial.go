package trustedappreleases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMalformedDenials = errors.New("the release denials are malformed")
	earliestEndTime     = time.Unix(1, 0)
)

type DenialSource interface {
	Denials(ctx context.Context) ([]Denial, error)
}

type Denial struct {
	EndTime            time.Time `json:"endTime"`
	SigningCertificate string    `json:"signingCertificate"`
}

func DenialsFrom(content []byte) ([]Denial, error) {
	var denials []Denial
	if err := json.Unmarshal(content, &denials); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedDenials, err)
	}
	for i, denial := range denials {
		denials[i].SigningCertificate = signingCertificateFrom(denial.SigningCertificate)
		if !isSigningCertificate(denials[i].SigningCertificate) ||
			denial.EndTime.Before(earliestEndTime) {
			return nil, fmt.Errorf("%w: %+v", ErrMalformedDenials, denial)
		}
	}

	return denials, nil
}

func signingCertificateFrom(writtenDigest string) string {
	return strings.ToLower(strings.ReplaceAll(writtenDigest, ":", ""))
}
