package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheSiteOfAnAddressDropsTheWorldWideWebLabel(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"https://www.kernel.org/":       "kernel.org",
		"https://kernel.org/":           "kernel.org",
		"http://WWW.Kernel.ORG/sub/dir": "kernel.org",
		"https://kernel.org./":          "kernel.org",
		"https://wwwtest.kernel.org/":   "wwwtest.kernel.org",
		"https://www.www.example.com/":  "www.example.com",
	}
	for address, want := range cases {
		if got := yacymodel.SiteOf(address); got != want {
			t.Errorf("the site of %q is %q, want %q", address, got, want)
		}
	}
}

func TestTheSiteOfAnAddressWithoutAHostIsTheAddress(t *testing.T) {
	t.Parallel()

	for _, address := range []string{"", "not an address", "mailto:someone@example.com"} {
		if got := yacymodel.SiteOf(address); got != address {
			t.Errorf("the site of %q is %q, want the address itself", address, got)
		}
	}
}
