package yacymodel

import (
	"errors"
	"fmt"
	"net/url"
)

var ErrBadSeedListURL = errors.New("bad seed list url")

type SeedListURL struct {
	value string
}

func ParseSeedListURL(s string) (SeedListURL, error) {
	parsed, err := url.Parse(s)
	if err != nil {
		return SeedListURL{}, fmt.Errorf("%w: %w", ErrBadSeedListURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return SeedListURL{}, fmt.Errorf("%w: scheme %q", ErrBadSeedListURL, parsed.Scheme)
	}
	if parsed.Host == "" {
		return SeedListURL{}, fmt.Errorf("%w: missing host", ErrBadSeedListURL)
	}
	return SeedListURL{value: s}, nil
}

func (u SeedListURL) String() string { return u.value }
