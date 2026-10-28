package yacyproto

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const addressSignaturePartSeparator = ":"

type addressSignatureWireCodec struct{}

func (addressSignatureWireCodec) encode(signature yacymodel.AddressSignature) string {
	return instantWireCodec{}.encode(signature.IssuedAt) + addressSignaturePartSeparator +
		yacymodel.Encode(signature.Signature[:])
}

func (addressSignatureWireCodec) decode(text string) (yacymodel.AddressSignature, bool) {
	issuedAtText, signatureText, separated := strings.Cut(text, addressSignaturePartSeparator)
	if !separated {
		return yacymodel.AddressSignature{}, false
	}
	issuedAt, ok := instantWireCodec{}.decode(issuedAtText)
	if !ok {
		return yacymodel.AddressSignature{}, false
	}
	signature, err := yacymodel.ParseSignature(signatureText)
	if err != nil {
		return yacymodel.AddressSignature{}, false
	}

	return yacymodel.AddressSignature{IssuedAt: issuedAt, Signature: signature}, true
}
