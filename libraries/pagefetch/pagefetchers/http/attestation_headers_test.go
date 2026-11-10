package http_test

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/optional"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	httppkg "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
)

var attestedHeaders = map[string]string{
	"Attested-Version":   "v0.1",
	"Attested-Statement": ":c3RhdGVtZW50:",
	"Attested-Signature": ":c2lnbmF0dXJl:",
	"Attested-Key":       ":a2V5:",
}

func TestFetchKeepsTheAttestationOfTheResponse(t *testing.T) {
	for name, status := range map[string]int{
		"page":     http.StatusOK,
		"redirect": http.StatusFound,
	} {
		attestation, attested := attestationServedWith(t, status, attestedHeaders, nil).Get()

		if !attested || string(attestation.Statement) != "statement" ||
			string(attestation.Signature) != "signature" ||
			string(attestation.KeyEvidence) != "key" {
			t.Errorf("%s: attestation = %+v, %v", name, attestation, attested)
		}
	}
}

func TestFetchKeepsNoAttestationOfAResponseWithBrokenAttestedHeaders(t *testing.T) {
	for name, change := range map[string]func(map[string]string){
		"no headers":        func(h map[string]string) { clear(h) },
		"other version":     func(h map[string]string) { h["Attested-Version"] = "v9" },
		"no statement":      func(h map[string]string) { delete(h, "Attested-Statement") },
		"unopened sequence": func(h map[string]string) { h["Attested-Key"] = "a2V5:" },
		"bad base64":        func(h map[string]string) { h["Attested-Signature"] = ":!!!:" },
		"empty sequence":    func(h map[string]string) { h["Attested-Statement"] = "::" },
	} {
		headers := map[string]string{}
		for header, value := range attestedHeaders {
			headers[header] = value
		}
		change(headers)

		if attestationServedWith(t, http.StatusOK, headers, nil).Present() {
			t.Errorf("%s: an attestation is kept", name)
		}
	}
}

func TestFetchKeepsNoAttestationOfAResponseWithAnAttestedHeaderTwice(t *testing.T) {
	repeated := map[string]string{"Attested-Statement": ":b3RoZXI=:"}

	if attestationServedWith(t, http.StatusOK, attestedHeaders, repeated).Present() {
		t.Error("an attestation is kept although its statement header comes twice")
	}
}

func attestationServedWith(
	t *testing.T,
	status int,
	headers map[string]string,
	addedHeaders map[string]string,
) optional.Value[pagefetch.ResponseAttestation] {
	t.Helper()
	proxy, closeFn := proxyURL(t, func(w http.ResponseWriter, _ *http.Request) {
		for header, value := range headers {
			w.Header().Set(header, value)
		}
		for header, value := range addedHeaders {
			w.Header().Add(header, value)
		}
		if status != http.StatusOK {
			w.Header().Set("Location", "http://target.example/b")
		}
		w.WriteHeader(status)
		_, _ = w.Write(bytes.Repeat([]byte("x"), 3))
	})
	defer closeFn()

	outcome, err := httppkg.New(
		httppkg.Proxy{
			URL:      proxy,
			DialMode: httppkg.ProxyDialTunnel,
			Dial:     (&net.Dialer{}).DialContext,
		},
		testUserAgent,
		1<<20,
		time.Second,
	).Fetch(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://target.example/a"),
		pagefetch.PageVersion{},
	)
	if err != nil {
		t.Fatal(err)
	}

	return outcome.Attestation
}
