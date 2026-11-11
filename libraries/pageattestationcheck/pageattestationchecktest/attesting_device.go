// Package pageattestationchecktest makes a fake attesting Android device and a fake time-stamp
// authority, so that tests can sign page attestations that the check accepts.
package pageattestationchecktest

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/digitorus/timestamp"
	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"
	"github.com/transparency-dev/merkle/rfc6962"
)

const (
	AppPackage            = "io.github.nikitakarpei.yacypeercompanion"
	AppSigningCertificate = "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"
	AppVersionCode        = 7
	deviceOSVersion       = 160000
	signDigestPurpose     = 2
	keyStorageVersion     = 300
	strongBoxLevel        = 2
	extendedKeyUsageID    = 37
)

type AttestingDevice struct {
	key   *ecdsa.PrivateKey
	chain [][]byte
	root  *x509.Certificate
}

func NewAttestingDevice(t testing.TB, versionCode int64) *AttestingDevice {
	t.Helper()
	now := time.Now()
	signer := mustHex(t, AppSigningCertificate)
	extension := keyDescriptionOf(AppPackage, versionCode, signer)
	serial := int64(0)
	certificate := func(
		subject pkix.Name,
		parent *x509.Certificate,
		parentKey *ecdsa.PrivateKey,
		extensions []pkix.Extension,
		authority bool,
	) (*x509.Certificate, *ecdsa.PrivateKey) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		serial++
		template := &x509.Certificate{
			SerialNumber: big.NewInt(now.UnixNano() + serial), Subject: subject,
			NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour),
			BasicConstraintsValid: true, IsCA: authority, ExtraExtensions: extensions,
		}
		if parent == nil {
			parent, parentKey = template, key
		}
		der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, parentKey)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}

		return parsed, key
	}
	root, rootKey := certificate(
		pkix.Name{CommonName: "test attestation root"},
		nil,
		nil,
		nil,
		true,
	)
	top, topKey := certificate(
		pkix.Name{
			CommonName:   "Droid CA2",
			Organization: []string{"Google LLC"},
		},
		root,
		rootKey,
		nil,
		true,
	)
	attestation, attestationKey := certificate(
		pkix.Name{CommonName: "f00d", Organization: []string{"StrongBox"}}, top, topKey, nil, true,
	)
	leaf, leafKey := certificate(
		pkix.Name{CommonName: "Android Keystore Key"}, attestation, attestationKey,
		[]pkix.Extension{{Id: android.OIDKeyAttestation, Value: extension}}, false,
	)

	return &AttestingDevice{
		key:   leafKey,
		chain: [][]byte{leaf.Raw, attestation.Raw, top.Raw},
		root:  root,
	}
}

func (d *AttestingDevice) Trust() *android.Trust {
	return &android.Trust{
		Roots:              []*x509.Certificate{d.root},
		Revoked:            map[string]string{},
		RevocationProduced: time.Now(),
		BootKeyLists: map[string]*android.BootKeyList{
			"grapheneos": {Keys: map[string]bool{}, Produced: time.Now()},
		},
	}
}

func (d *AttestingDevice) KeyEvidence() []byte {
	evidence, _ := json.Marshal(map[string]any{ //nolint:errchkjson // a name and bytes.
		"profile": android.Name, "certificates": d.chain,
	})

	return evidence
}

func (d *AttestingDevice) PublicKeyInfo() []byte {
	certificate, _ := x509.ParseCertificate(d.chain[0])

	return certificate.RawSubjectPublicKeyInfo
}

func (d *AttestingDevice) Sign(message []byte) ([]byte, error) {
	digest := sha256.Sum256(message)
	signature, err := ecdsa.SignASN1(rand.Reader, d.key, digest[:])
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}

	return lowS(signature), nil
}

func (d *AttestingDevice) BundleOf(t testing.TB, statement []byte) *verifier.Bundle {
	t.Helper()
	signature, err := d.Sign(verifier.PAE(verifier.InTotoPayload, statement))
	if err != nil {
		t.Fatal(err)
	}
	var key verifier.KeyEvidence
	if err := key.UnmarshalJSON(d.KeyEvidence()); err != nil {
		t.Fatal(err)
	}

	return &verifier.Bundle{
		MediaType: verifier.BundleMediaType,
		DSSEEnvelope: verifier.DSSEEnvelope{
			PayloadType: verifier.InTotoPayload,
			Payload:     statement,
			Signatures:  []verifier.DSSESignature{{Sig: signature}},
		},
		VerificationMaterial: verifier.VerificationMaterial{Key: key},
	}
}

func lowS(signature []byte) []byte {
	var parts struct{ R, S *big.Int }
	_, _ = asn1.Unmarshal(signature, &parts)
	order := elliptic.P256().Params().N
	if parts.S.Cmp(new(big.Int).Rsh(order, 1)) > 0 {
		parts.S = new(big.Int).Sub(order, parts.S)
	}
	low, _ := asn1.Marshal(parts)

	return low
}

func keyDescriptionOf(pkg string, versionCode int64, signer []byte) []byte {
	rootOfTrust := sequence(
		octets(
			make([]byte, 32),
		),
		tagged(0, 1, false, []byte{0xff}),
		enumerated(0),
		octets(make([]byte, 32)),
	)
	applicationID := sequence(
		set(sequence(octets([]byte(pkg)), integer(versionCode))), set(octets(signer)),
	)
	patchMonth := int64(time.Now().UTC().Year()*100 + int(time.Now().UTC().Month()))
	hardwareEnforced := concatenated(
		explicit(
			1,
			set(integer(signDigestPurpose)),
		),
		explicit(2, integer(3)),
		explicit(5, set(integer(4))),
		explicit(10, integer(1)),
		explicit(702, integer(0)),
		explicit(704, rootOfTrust),
		explicit(705, integer(deviceOSVersion)),
		explicit(706, integer(patchMonth)),
		explicit(718, integer(patchMonth*100+5)),
		explicit(719, integer(patchMonth*100+5)),
	)
	softwareEnforced := sequence(
		explicit(701, integer(1762130000000)),
		explicit(709, octets(applicationID)),
	)

	return sequence(
		integer(keyStorageVersion), enumerated(strongBoxLevel), integer(keyStorageVersion),
		enumerated(strongBoxLevel), octets([]byte(android.KeyChallenge)), octets(nil),
		softwareEnforced, tagged(0, 16, true, hardwareEnforced),
	)
}

func tagged(class, tag int, constructed bool, content []byte) []byte {
	identifier := byte((class & 0x03) << 6)
	if constructed {
		identifier |= 0x20
	}
	var encoding []byte
	if tag < 31 {
		encoding = []byte{identifier | byte(tag&0x1f)}
	} else {
		var tagBytes []byte
		for rest := tag; rest > 0; rest >>= 7 {
			tagBytes = append([]byte{byte(rest&0x7f) | 0x80}, tagBytes...)
		}
		tagBytes[len(tagBytes)-1] &= 0x7f
		encoding = append([]byte{identifier | 0x1f}, tagBytes...)
	}
	switch length := len(content); {
	case length < 128:
		encoding = append(encoding, byte(length))
	case length < 256:
		encoding = append(encoding, 0x81, byte(length))
	default:
		encoding = append(encoding, 0x82, byte((length>>8)&0xff), byte(length&0xff))
	}

	return append(encoding, content...)
}

func concatenated(parts ...[]byte) []byte {
	var encoding []byte
	for _, part := range parts {
		encoding = append(encoding, part...)
	}

	return encoding
}

func sequence(parts ...[]byte) []byte       { return tagged(0, 16, true, concatenated(parts...)) }
func set(parts ...[]byte) []byte            { return tagged(0, 17, true, concatenated(parts...)) }
func octets(content []byte) []byte          { return tagged(0, 4, false, content) }
func explicit(tag int, value []byte) []byte { return tagged(2, tag, true, value) }
func enumerated(value int64) []byte         { return tagged(0, 10, false, integer(value)[2:]) }

func integer(value int64) []byte {
	encoding := big.NewInt(value).Bytes()
	if len(encoding) == 0 || encoding[0]&0x80 != 0 {
		encoding = append([]byte{0}, encoding...)
	}

	return tagged(0, 2, false, encoding)
}

type TimeStampAuthority struct {
	certificate *x509.Certificate
	key         *ecdsa.PrivateKey
	root        *x509.Certificate
}

func NewTimeStampAuthority(t testing.TB) *TimeStampAuthority {
	t.Helper()
	now := time.Now()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test TSA root"},
		NotBefore: now.Add(-90 * 24 * time.Hour), NotAfter: now.Add(48 * time.Hour),
		BasicConstraintsValid: true, IsCA: true, KeyUsage: x509.KeyUsageCertSign,
	}
	rootDER, err := x509.CreateCertificate(
		rand.Reader,
		rootTemplate,
		rootTemplate,
		&rootKey.PublicKey,
		rootKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	usage, err := asn1.Marshal([]asn1.ObjectIdentifier{{1, 3, 6, 1, 5, 5, 7, 3, 8}})
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "test TSA"},
		NotBefore: now.Add(-90 * 24 * time.Hour), NotAfter: now.Add(48 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
		ExtraExtensions: []pkix.Extension{{
			Id: asn1.ObjectIdentifier{2, 5, 29, extendedKeyUsageID}, Critical: true, Value: usage,
		}},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, root, &key.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}

	return &TimeStampAuthority{certificate: certificate, key: key, root: root}
}

func (a *TimeStampAuthority) policyEntry() json.RawMessage {
	pemOf := func(certificate *x509.Certificate) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw}))
	}
	entry, _ := json.Marshal(map[string]any{ //nolint:errchkjson // strings only.
		"name":                "test TSA",
		"root":                pemOf(a.root),
		"signingCertificates": []string{pemOf(a.certificate)},
	})

	return entry
}

func (a *TimeStampAuthority) Stamp(t testing.TB, bundle *verifier.Bundle, at time.Time) {
	t.Helper()
	binding, err := android.New(nil).Binding(bundle.VerificationMaterial.Key.Raw)
	if err != nil {
		t.Fatal(err)
	}
	signedStatementID := verifier.SignedStatementID(
		android.Name, verifier.StatementID(bundle.DSSEEnvelope.Payload), binding,
		bundle.DSSEEnvelope.Signatures[0].Sig,
	)
	bundle.VerificationMaterial.Timestamp = a.timestampOf(t, signedStatementID[:], at)
}

func (a *TimeStampAuthority) TimeStampEncodingOf(
	t testing.TB,
	signedStatementID []byte,
	at time.Time,
) []byte {
	t.Helper()
	encoding, err := json.Marshal(a.timestampOf(t, signedStatementID, at))
	if err != nil {
		t.Fatal(err)
	}

	return encoding
}

func (a *TimeStampAuthority) timestampOf(
	t testing.TB,
	signedStatementID []byte,
	at time.Time,
) *verifier.Timestamp {
	t.Helper()
	token, err := (&timestamp.Timestamp{
		HashAlgorithm: crypto.SHA256,
		HashedMessage: rfc6962.DefaultHasher.HashLeaf(signedStatementID),
		Time:          at,
		Policy:        asn1.ObjectIdentifier{1, 2, 3, 4},
	}).CreateResponseWithOpts(a.certificate, a.key, crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}

	return &verifier.Timestamp{
		Token: token,
		Tree:  verifier.Tree{Size: 1, Index: 0, Path: []verifier.Bytes{}},
	}
}

func mustHex(t testing.TB, encoded string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}

	return decoded
}
