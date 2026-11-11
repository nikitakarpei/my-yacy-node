package pagescrapecontract_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestPageOfferMessageOfHoldsTheEncodedPageWithoutAttestation(t *testing.T) {
	page := pagescrapecontract.OfferedPage{
		PageURL:   canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		LandedURL: canonicalurltest.CanonicalURLOf(t, "https://example.org/a"),
		Body:      []byte("page"),
	}

	message := pagescrapecontract.PageOfferMessageOf(page)
	read, err := pagescrapecontract.UnmarshalOfferedPage(message.Body)
	if err != nil {
		t.Fatal(err)
	}

	if read.PageURL != page.PageURL || string(read.Body) != "page" ||
		len(message.Attestation) != 0 {
		t.Errorf("message = %+v", message)
	}
}

func TestAnAttestedPageOfferMessageKeepsItsBody(t *testing.T) {
	message := pagescrapecontract.PageOfferMessage{
		Body: []byte("body"),
	}.AttestedBy(
		[]byte("bundle"),
	)

	if string(message.Attestation) != "bundle" || string(message.Body) != "body" {
		t.Errorf("message = %+v", message)
	}
}

func TestPageOfferMessageFromAHeaderIsAttestedOnlyWhenTheHeaderIsGiven(t *testing.T) {
	attested := pagescrapecontract.PageOfferMessageFrom([]byte("body"), "bundle")
	unattested := pagescrapecontract.PageOfferMessageFrom([]byte("body"), "")

	if string(attested.Attestation) != "bundle" {
		t.Errorf("attested message = %+v", attested)
	}
	if len(unattested.Attestation) != 0 || string(unattested.Body) != "body" {
		t.Errorf("unattested message = %+v", unattested)
	}
}
