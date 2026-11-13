package trustedappreleases_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

func TestDenialsNameASigningCertificateAndAnEndTime(t *testing.T) {
	for name, test := range map[string]struct {
		content string
		want    []trustedappreleases.Denial
	}{
		"one denial": {
			`[{"signingCertificate":"` + signingCertificate + `","endTime":"2026-11-13T10:00:00Z"}]`,
			[]trustedappreleases.Denial{{
				SigningCertificate: signingCertificate,
				EndTime:            time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC),
			}},
		},
		"keytool digest": {
			`[{"signingCertificate":"` + keytoolDigestOf(signingCertificate) + `","endTime":"2026-11-13T10:00:00Z"}]`,
			[]trustedappreleases.Denial{{
				SigningCertificate: signingCertificate,
				EndTime:            time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC),
			}},
		},
		"empty list": {`[]`, []trustedappreleases.Denial{}},
	} {
		denials, err := trustedappreleases.DenialsFrom([]byte(test.content))

		if err != nil || !slices.Equal(denials, test.want) {
			t.Errorf("%s: denials = %+v, %v, want %+v", name, denials, err, test.want)
		}
	}
}

func TestMalformedDenialsDenyNoRelease(t *testing.T) {
	for name, malformed := range map[string]string{
		"not JSON":          "{",
		"short certificate": `[{"signingCertificate":"72e2","endTime":"2026-11-13T10:00:00Z"}]`,
		"no end time":       `[{"signingCertificate":"` + signingCertificate + `"}]`,
		"end time of 1970":  `[{"signingCertificate":"` + signingCertificate + `","endTime":"1970-01-01T00:00:00Z"}]`,
	} {
		_, err := trustedappreleases.DenialsFrom([]byte(malformed))

		if !errors.Is(err, trustedappreleases.ErrMalformedDenials) {
			t.Errorf("%s: error = %v, want %v", name, err, trustedappreleases.ErrMalformedDenials)
		}
	}
}

func keytoolDigestOf(digest string) string {
	pairs := make([]string, 0, len(digest)/2)
	for i := 0; i < len(digest); i += 2 {
		pairs = append(pairs, strings.ToUpper(digest[i:i+2]))
	}

	return strings.Join(pairs, ":")
}
