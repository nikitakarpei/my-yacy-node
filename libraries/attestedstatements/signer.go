package attestedstatements

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json/v2"
	"errors"
	"fmt"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
)

var errNotAnECKey = errors.New("the hardware key is not an EC key")

type Signer struct {
	keys               HardwareKeys
	timeStampAuthority TimeStampAuthority
	profile            verifier.Profile
	observer           SigningObserver
}

func New(
	keys HardwareKeys,
	timeStampAuthority TimeStampAuthority,
	profile verifier.Profile,
	observer SigningObserver,
) Signer {
	return Signer{
		keys:               keys,
		timeStampAuthority: timeStampAuthority,
		profile:            profile,
		observer:           observer,
	}
}

func (s Signer) Sign(ctx context.Context, statement []byte, stamping TimeStamping) ([]byte, bool) {
	bundle, signed := s.sign(ctx, statement)
	if !signed {
		return nil, false
	}
	if stamping == TimeStamped {
		bundle, signed = s.stamp(ctx, bundle)
		if !signed {
			return nil, false
		}
	}
	return encodingOf(bundle), true
}

func encodingOf(bundle verifier.Bundle) []byte {
	encoding, _ := json.Marshal(bundle) //nolint:errchkjson // bytes, strings and raw JSON only.

	return encoding
}

func (s Signer) sign(ctx context.Context, statement []byte) (verifier.Bundle, bool) {
	key, err := s.keys.CurrentKey()
	if err != nil {
		s.observer.KeyUnavailable(ctx, err)

		return verifier.Bundle{}, false
	}
	signature, err := key.Sign(verifier.PAE(verifier.InTotoPayload, statement))
	if err != nil {
		s.observer.SignatureRefused(ctx, err)

		return verifier.Bundle{}, false
	}
	var evidence verifier.KeyEvidence
	if err := evidence.UnmarshalJSON(key.KeyEvidence()); err != nil {
		s.observer.KeyEvidenceUnreadable(ctx, err)

		return verifier.Bundle{}, false
	}

	return bundleFrom(statement, signature, evidence), true
}

func bundleFrom(statement, signature []byte, evidence verifier.KeyEvidence) verifier.Bundle {
	return verifier.Bundle{
		MediaType: verifier.BundleMediaType,
		DSSEEnvelope: verifier.DSSEEnvelope{
			PayloadType: verifier.InTotoPayload,
			Payload:     statement,
			Signatures:  []verifier.DSSESignature{{Sig: signature}},
		},
		VerificationMaterial: verifier.VerificationMaterial{Key: evidence},
	}
}

func (s Signer) stamp(ctx context.Context, bundle verifier.Bundle) (verifier.Bundle, bool) {
	binding, err := s.profile.Binding(bundle.VerificationMaterial.Key.Raw)
	if err != nil {
		s.observer.KeyEvidenceUnreadable(ctx, err)

		return verifier.Bundle{}, false
	}
	signedStatementID := verifier.SignedStatementID(
		s.profile.Name(),
		verifier.StatementID(bundle.DSSEEnvelope.Payload),
		binding,
		bundle.DSSEEnvelope.Signatures[0].Sig,
	)
	encoding, err := s.timeStampAuthority.Stamp(signedStatementID[:])
	if err != nil {
		s.observer.TimeStampUnavailable(ctx, err)

		return verifier.Bundle{}, false
	}
	var timestamp verifier.Timestamp
	if err := verifier.UnmarshalStrict(encoding, &timestamp); err != nil {
		s.observer.TimeStampUnavailable(ctx, fmt.Errorf("read time-stamp: %w", err))

		return verifier.Bundle{}, false
	}
	bundle.VerificationMaterial.Timestamp = &timestamp

	return bundle, true
}

func (s Signer) IsOwnSignature(ctx context.Context, statement, signature []byte) bool {
	publicKey, available := s.currentPublicKey(ctx)
	if !available {
		return false
	}
	digest := sha256.Sum256(verifier.PAE(verifier.InTotoPayload, statement))

	return ecdsa.VerifyASN1(publicKey, digest[:], signature)
}

func (s Signer) currentPublicKey(ctx context.Context) (*ecdsa.PublicKey, bool) {
	key, err := s.keys.CurrentKey()
	if err != nil {
		s.observer.KeyUnavailable(ctx, err)

		return nil, false
	}
	publicKey, err := x509.ParsePKIXPublicKey(key.PublicKeyInfo())
	if err != nil {
		s.observer.KeyEvidenceUnreadable(ctx, err)

		return nil, false
	}
	ecKey, isEC := publicKey.(*ecdsa.PublicKey)
	if !isEC {
		s.observer.KeyEvidenceUnreadable(ctx, errNotAnECKey)

		return nil, false
	}

	return ecKey, true
}
