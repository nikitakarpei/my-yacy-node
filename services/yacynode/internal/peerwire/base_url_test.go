package peerwire_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

func TestParseSchemeKnowsHTTPAndHTTPSOnly(t *testing.T) {
	for _, text := range []string{"http", "https"} {
		if scheme, err := peerwire.ParseScheme(text); err != nil || string(scheme) != text {
			t.Errorf("ParseScheme(%q) = %q, %v", text, scheme, err)
		}
	}
	if _, err := peerwire.ParseScheme("ftp"); err == nil {
		t.Error("ParseScheme(ftp) accepted an unknown scheme")
	}
}
