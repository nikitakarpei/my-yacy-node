package http_test

import (
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases/denialsources/http"
)

const signingCertificate = "72e21ab017f94f81137bb0d983453d02cfb59b31eac7935c31473a3ad2b2c428"

func TestTheDenialsAreThoseOfTheDocument(t *testing.T) {
	address := serve(t, nethttp.StatusOK,
		`[{"signingCertificate":"`+signingCertificate+`","endTime":"2026-11-13T10:00:00Z"}]`)

	denials, err := http.New(nethttp.DefaultClient, address).Denials(t.Context())

	if err != nil || len(denials) != 1 || denials[0].SigningCertificate != signingCertificate {
		t.Errorf("denials = %+v, %v", denials, err)
	}
}

func TestAMalformedDocumentGivesNoDenials(t *testing.T) {
	address := serve(t, nethttp.StatusOK, "{")

	_, err := http.New(nethttp.DefaultClient, address).Denials(t.Context())

	if !errors.Is(err, trustedappreleases.ErrMalformedDenials) {
		t.Errorf("error = %v, want %v", err, trustedappreleases.ErrMalformedDenials)
	}
}

func TestAMissingDocumentGivesNoDenials(t *testing.T) {
	address := serve(t, nethttp.StatusNotFound, "")

	_, err := http.New(nethttp.DefaultClient, address).Denials(t.Context())

	if !errors.Is(err, http.ErrUnexpectedStatus) {
		t.Errorf("error = %v, want %v", err, http.ErrUnexpectedStatus)
	}
}

func TestADocumentThatIsTooLargeGivesNoDenials(t *testing.T) {
	address := serve(t, nethttp.StatusOK, strings.Repeat(" ", 1<<20+1))

	_, err := http.New(nethttp.DefaultClient, address).Denials(t.Context())

	if !errors.Is(err, http.ErrTooLarge) {
		t.Errorf("error = %v, want %v", err, http.ErrTooLarge)
	}
}

func TestAnUnreachableAddressGivesNoDenials(t *testing.T) {
	server := httptest.NewServer(nethttp.NotFoundHandler())
	server.Close()

	if _, err := http.New(nethttp.DefaultClient, server.URL).Denials(t.Context()); err == nil {
		t.Error("an unreachable address gave denials")
	}
}

func serve(t *testing.T, status int, document string) string {
	t.Helper()
	server := httptest.NewServer(
		nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(document))
		}),
	)
	t.Cleanup(server.Close)

	return server.URL
}
