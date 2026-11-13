package trustedappreleases

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrMalformedRecord = errors.New("the release record is malformed")

type TrustedRelease struct {
	DeniedFrom         time.Time `json:"deniedFrom,omitzero"`
	Tag                string    `json:"tag"`
	SigningCertificate string    `json:"signingCertificate"`
	VersionCode        int64     `json:"versionCode"`
}

type releaseRecord struct {
	SigningCertificate string `json:"signingCertificate"`
	VersionCode        int64  `json:"versionCode"`
}

func trustedReleaseFrom(tag string, record []byte) (TrustedRelease, error) {
	var parsed releaseRecord
	if err := json.Unmarshal(record, &parsed); err != nil {
		return TrustedRelease{}, fmt.Errorf("%w: %w", ErrMalformedRecord, err)
	}
	if !isSigningCertificate(parsed.SigningCertificate) || parsed.VersionCode <= 0 {
		return TrustedRelease{}, ErrMalformedRecord
	}

	return TrustedRelease{
		Tag:                tag,
		SigningCertificate: parsed.SigningCertificate,
		VersionCode:        parsed.VersionCode,
	}, nil
}

func isSigningCertificate(signingCertificate string) bool {
	digest, err := hex.DecodeString(signingCertificate)

	return err == nil && len(digest) == sha256.Size &&
		hex.EncodeToString(digest) == signingCertificate
}
