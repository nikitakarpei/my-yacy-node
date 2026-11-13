package trustedappreleases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrMalformedDenials = errors.New("the denied releases are malformed")

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
	for _, denial := range denials {
		if !isSigningCertificate(denial.SigningCertificate) || denial.EndTime.IsZero() {
			return nil, fmt.Errorf("%w: %+v", ErrMalformedDenials, denial)
		}
	}

	return denials, nil
}
