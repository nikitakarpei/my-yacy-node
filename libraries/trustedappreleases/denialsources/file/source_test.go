package file_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases/denialsources/file"
)

const signingCertificate = "72e21ab017f94f81137bb0d983453d02cfb59b31eac7935c31473a3ad2b2c428"

func TestTheDenialsAreThoseOfTheFile(t *testing.T) {
	path := writtenPath(
		t,
		`[{"signingCertificate":"`+signingCertificate+`","endTime":"2026-11-13T10:00:00Z"}]`,
	)

	denials, err := file.New(path).Denials(t.Context())

	if err != nil || len(denials) != 1 || denials[0].SigningCertificate != signingCertificate {
		t.Errorf("denials = %+v, %v", denials, err)
	}
}

func TestAMissingFileGivesNoDenials(t *testing.T) {
	_, err := file.New(filepath.Join(t.TempDir(), "missing.json")).Denials(t.Context())

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want %v", err, os.ErrNotExist)
	}
}

func writtenPath(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "release-denials.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}
