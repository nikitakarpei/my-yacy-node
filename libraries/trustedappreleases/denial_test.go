package trustedappreleases_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

func TestDenialsNameASigningCertificateAndAnEndTime(t *testing.T) {
	denials, err := trustedappreleases.DenialsFrom([]byte(
		`[{"signingCertificate":"` + signingCertificate + `","endTime":"2026-11-13T10:00:00Z"}]`,
	))
	if err != nil {
		t.Fatal(err)
	}

	want := []trustedappreleases.Denial{{
		SigningCertificate: signingCertificate,
		EndTime:            time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC),
	}}
	if !slices.Equal(denials, want) {
		t.Errorf("denials = %+v, want %+v", denials, want)
	}
}

func TestAnEmptyListDeniesNoRelease(t *testing.T) {
	denials, err := trustedappreleases.DenialsFrom([]byte(`[]`))

	if err != nil || len(denials) != 0 {
		t.Errorf("denials = %+v, %v, want none", denials, err)
	}
}

func TestMalformedDenialsDenyNoRelease(t *testing.T) {
	for name, malformed := range map[string]string{
		"not JSON":          "{",
		"short certificate": `[{"signingCertificate":"72e2","endTime":"2026-11-13T10:00:00Z"}]`,
		"no end time":       `[{"signingCertificate":"` + signingCertificate + `"}]`,
	} {
		_, err := trustedappreleases.DenialsFrom([]byte(malformed))

		if !errors.Is(err, trustedappreleases.ErrMalformedDenials) {
			t.Errorf("%s: error = %v, want %v", name, err, trustedappreleases.ErrMalformedDenials)
		}
	}
}
