package pagesite_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/pagesite"
)

func TestRegistrableDomainOf(t *testing.T) {
	for address, wantedDomain := range map[string]string{
		"https://www.Example.co.uk/page": "example.co.uk",
		"https://blog.example.com/":      "example.com",
		"https://someone.blogspot.com/":  "someone.blogspot.com",
		"http://192.168.0.1/admin":       "192.168.0.1",
		"http://[2001:db8::1]:8080/":     "2001:db8::1",
		"https://co.uk/":                 "co.uk",
	} {
		t.Run(address, func(t *testing.T) {
			canonical, err := canonicalurl.CanonicalURLOf(address)
			if err != nil {
				t.Fatal(err)
			}
			if domain := pagesite.RegistrableDomainOf(canonical); domain != wantedDomain {
				t.Fatalf("registrable domain of %s is %q, want %q", address, domain, wantedDomain)
			}
		})
	}
}
