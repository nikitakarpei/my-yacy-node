package pagescrapecontract

import "github.com/nikitakarpei/yacy-rwi-node/optional"

const PageOfferAttestationHeader = "Page-Offer-Attestation"

type PageOfferMessage struct {
	Body        []byte
	Attestation optional.Value[[]byte]
}

func PageOfferMessageOf(page OfferedPage) PageOfferMessage {
	body, _ := MarshalOfferedPage(page)

	return PageOfferMessage{Body: body}
}

func PageOfferMessageFrom(body []byte, attestationHeader string) PageOfferMessage {
	if attestationHeader == "" {
		return PageOfferMessage{Body: body}
	}

	return PageOfferMessage{Body: body, Attestation: optional.Some([]byte(attestationHeader))}
}

func (m PageOfferMessage) AttestedBy(bundle []byte) PageOfferMessage {
	m.Attestation = optional.Some(bundle)

	return m
}
