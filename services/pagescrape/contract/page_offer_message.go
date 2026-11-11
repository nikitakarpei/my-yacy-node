package pagescrapecontract

import "github.com/nikitakarpei/yacy-rwi-node/attestedstatements"

const PageOfferAttestationHeader = "Page-Offer-Attestation"

type PageOfferMessage struct {
	Body        []byte
	Attestation attestedstatements.Bundle
}

func PageOfferMessageOf(page OfferedPage) PageOfferMessage {
	body, _ := MarshalOfferedPage(page)

	return PageOfferMessage{Body: body}
}

func PageOfferMessageFrom(body []byte, attestationHeader string) PageOfferMessage {
	return PageOfferMessage{Body: body, Attestation: attestedstatements.Bundle(attestationHeader)}
}

func (m PageOfferMessage) AttestedBy(bundle attestedstatements.Bundle) PageOfferMessage {
	m.Attestation = bundle

	return m
}
