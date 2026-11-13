package trustedappreleases

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrMalformedRecord = errors.New("the release record is malformed")

type TrustedRelease struct {
	Tag                string
	SigningCertificate string
	VersionCode        int64
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
	digest, err := hex.DecodeString(parsed.SigningCertificate)
	if err != nil || len(digest) != sha256.Size ||
		hex.EncodeToString(digest) != parsed.SigningCertificate ||
		parsed.VersionCode <= 0 {
		return TrustedRelease{}, ErrMalformedRecord
	}

	return TrustedRelease{
		Tag:                tag,
		SigningCertificate: parsed.SigningCertificate,
		VersionCode:        parsed.VersionCode,
	}, nil
}
