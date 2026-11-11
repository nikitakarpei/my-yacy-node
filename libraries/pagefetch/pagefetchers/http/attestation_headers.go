package http

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/optional"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

const (
	headerAttestedVersion   = "Attested-Version"
	headerAttestedStatement = "Attested-Statement"
	headerAttestedSignature = "Attested-Signature"
	headerAttestedKey       = "Attested-Key"
	attestedFormatVersion   = "v0.1"
	byteSequenceDelimiter   = ":"
)

func attestationOf(header http.Header) optional.Value[pagefetch.ResponseAttestation] {
	if version, given := soleValueOf(header, headerAttestedVersion); !given ||
		version != attestedFormatVersion {
		return optional.None[pagefetch.ResponseAttestation]()
	}
	statement, statementGiven := byteSequenceOf(header, headerAttestedStatement)
	signature, signatureGiven := byteSequenceOf(header, headerAttestedSignature)
	keyEvidence, keyEvidenceGiven := byteSequenceOf(header, headerAttestedKey)
	if !statementGiven || !signatureGiven || !keyEvidenceGiven {
		return optional.None[pagefetch.ResponseAttestation]()
	}

	return optional.Some(pagefetch.ResponseAttestation{
		Statement: statement, Signature: signature, KeyEvidence: keyEvidence,
	})
}

func soleValueOf(header http.Header, name string) (string, bool) {
	values := header.Values(name)
	if len(values) != 1 {
		return "", false
	}

	return strings.TrimSpace(values[0]), true
}

func byteSequenceOf(header http.Header, name string) ([]byte, bool) {
	value, given := soleValueOf(header, name)
	encoded, opened := strings.CutPrefix(value, byteSequenceDelimiter)
	encoded, closed := strings.CutSuffix(encoded, byteSequenceDelimiter)
	if !given || !opened || !closed {
		return nil, false
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(decoded) == 0 {
		return nil, false
	}

	return decoded, true
}
