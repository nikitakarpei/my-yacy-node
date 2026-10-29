package peertls

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const endorsementScheme = "yacy-tls-key"

const endorsementPartSeparator = ":"

var (
	errNoEndorsement        = errors.New("no TLS key endorsement")
	errMalformedEndorsement = errors.New("malformed TLS key endorsement")
)

func endorsementURIOf(endorsement yacymodel.TLSKeyEndorsement) *url.URL {
	return &url.URL{
		Scheme: endorsementScheme,
		Opaque: endorsement.Identity.String() + endorsementPartSeparator +
			yacymodel.Encode(endorsement.Signature[:]),
	}
}

func endorsementOf(certificate *x509.Certificate) (yacymodel.TLSKeyEndorsement, error) {
	for _, uri := range certificate.URIs {
		if uri.Scheme == endorsementScheme {
			return endorsementFrom(uri.Opaque)
		}
	}

	return yacymodel.TLSKeyEndorsement{}, errNoEndorsement
}

func endorsementFrom(text string) (yacymodel.TLSKeyEndorsement, error) {
	cut := strings.LastIndex(text, endorsementPartSeparator)
	if cut < 0 {
		return yacymodel.TLSKeyEndorsement{}, errMalformedEndorsement
	}
	identity, err := yacymodel.ParsePeerIdentity(text[:cut])
	if err != nil {
		return yacymodel.TLSKeyEndorsement{}, fmt.Errorf("%w: %w", errMalformedEndorsement, err)
	}
	signature, err := yacymodel.ParseSignature(text[cut+1:])
	if err != nil {
		return yacymodel.TLSKeyEndorsement{}, fmt.Errorf("%w: %w", errMalformedEndorsement, err)
	}

	return yacymodel.TLSKeyEndorsement{Identity: identity, Signature: signature}, nil
}
