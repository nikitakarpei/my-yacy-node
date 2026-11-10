package statementsigning_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/statementsigning"
)

var (
	statement     = []byte(`{"_type":"https://in-toto.io/Statement/v1"}`)
	errNoKey      = errors.New("no key")
	errNoStamp    = errors.New("no time-stamp")
	errNoSigning  = errors.New("refused")
	stampEncoding = []byte(`{"token":"AQID","tree":{"size":1,"index":0,"path":[]}}`)
)

func TestBundleOfHoldsTheStatementSignedByTheCurrentKey(t *testing.T) {
	keys := newFakeKeys(t)
	signer := signerOver(keys)

	encoding, signed := signer.BundleOf(context.Background(), statement)
	if !signed {
		t.Fatal("statement not signed")
	}
	bundle := parsedBundle(t, encoding)

	if string(bundle.DSSEEnvelope.Payload) != string(statement) {
		t.Errorf("payload = %s, want %s", bundle.DSSEEnvelope.Payload, statement)
	}
	if bundle.VerificationMaterial.Timestamp != nil {
		t.Error("an unstamped bundle holds a time-stamp")
	}
	if !signer.HasSigned(context.Background(), statement, bundle.DSSEEnvelope.Signatures[0].Sig) {
		t.Error("the signer does not recognize its own bundle")
	}
}

func TestStampedBundleOfHoldsTheTimeStampOfItsSignedStatementID(t *testing.T) {
	keys := newFakeKeys(t)

	encoding, signed := signerOver(keys).StampedBundleOf(context.Background(), statement)
	if !signed {
		t.Fatal("statement not signed")
	}
	bundle := parsedBundle(t, encoding)

	if bundle.VerificationMaterial.Timestamp == nil ||
		string(bundle.VerificationMaterial.Timestamp.Token) != "\x01\x02\x03" {
		t.Errorf(
			"time-stamp = %+v, want the one the keys gave",
			bundle.VerificationMaterial.Timestamp,
		)
	}
	want := verifier.SignedStatementID(
		android.Name,
		verifier.StatementID(statement),
		keys.certificate.RawTBSCertificate,
		bundle.DSSEEnvelope.Signatures[0].Sig,
	)
	if string(keys.stampedID) != string(want[:]) {
		t.Errorf("stamped id = %x, want %x", keys.stampedID, want)
	}
}

func TestHasSignedRefusesTheSignatureOfAnotherKey(t *testing.T) {
	encoding, _ := signerOver(newFakeKeys(t)).BundleOf(context.Background(), statement)
	signature := parsedBundle(t, encoding).DSSEEnvelope.Signatures[0].Sig

	if signerOver(newFakeKeys(t)).HasSigned(context.Background(), statement, signature) {
		t.Error("a signer recognizes the signature of another key")
	}
}

func TestHasSignedRefusesAnotherStatement(t *testing.T) {
	keys := newFakeKeys(t)
	encoding, _ := signerOver(keys).BundleOf(context.Background(), statement)
	signature := parsedBundle(t, encoding).DSSEEnvelope.Signatures[0].Sig

	if signerOver(keys).HasSigned(context.Background(), []byte("{}"), signature) {
		t.Error("a signature of one statement is recognized for another")
	}
}

func TestNothingIsSignedWhenTheKeysFail(t *testing.T) {
	for name, breakKeys := range map[string]func(*fakeKeys){
		"no current key":        func(k *fakeKeys) { k.currentKeyErr = errNoKey },
		"signature refused":     func(k *fakeKeys) { k.signErr = errNoSigning },
		"unreadable evidence":   func(k *fakeKeys) { k.evidence = []byte("{}") },
		"no time-stamp":         func(k *fakeKeys) { k.stampErr = errNoStamp },
		"unreadable time-stamp": func(k *fakeKeys) { k.stamp = []byte(`{"token":1}`) },
	} {
		keys := newFakeKeys(t)
		breakKeys(keys)
		if _, signed := signerOver(keys).StampedBundleOf(context.Background(), statement); signed {
			t.Errorf("%s: statement signed", name)
		}
	}
}

func TestHasSignedRefusesWhenTheCurrentKeyIsUnknown(t *testing.T) {
	keys := newFakeKeys(t)
	encoding, _ := signerOver(keys).BundleOf(context.Background(), statement)
	signature := parsedBundle(t, encoding).DSSEEnvelope.Signatures[0].Sig
	for name, breakKeys := range map[string]func(*fakeKeys){
		"no current key":        func(k *fakeKeys) { k.currentKeyErr = errNoKey },
		"unreadable key":        func(k *fakeKeys) { k.publicKeyInfo = []byte("no key") },
		"key that is no EC key": func(k *fakeKeys) { k.publicKeyInfo = ed25519KeyInfo(t) },
	} {
		broken := *keys
		breakKeys(&broken)
		if signerOver(&broken).HasSigned(context.Background(), statement, signature) {
			t.Errorf("%s: the bundle is recognized", name)
		}
	}
}

type fakeKeys struct {
	privateKey    *ecdsa.PrivateKey
	certificate   *x509.Certificate
	evidence      []byte
	publicKeyInfo []byte
	stamp         []byte
	stampedID     []byte
	currentKeyErr error
	signErr       error
	stampErr      error
}

func newFakeKeys(t *testing.T) *fakeKeys {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "hardware key"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := json.Marshal(map[string]any{
		"profile":      android.Name,
		"certificates": []string{base64.StdEncoding.EncodeToString(der)},
	})
	if err != nil {
		t.Fatal(err)
	}

	return &fakeKeys{
		privateKey:    privateKey,
		certificate:   certificate,
		evidence:      evidence,
		publicKeyInfo: certificate.RawSubjectPublicKeyInfo,
		stamp:         stampEncoding,
	}
}

func (k *fakeKeys) CurrentKey() (statementsigning.HardwareKey, error) {
	if k.currentKeyErr != nil {
		return nil, k.currentKeyErr
	}

	return k, nil
}

func (k *fakeKeys) TimeStampOf(signedStatementID []byte) ([]byte, error) {
	k.stampedID = signedStatementID

	return k.stamp, k.stampErr
}

func (k *fakeKeys) KeyEvidence() []byte {
	return k.evidence
}

func (k *fakeKeys) PublicKeyInfo() []byte {
	return k.publicKeyInfo
}

func (k *fakeKeys) Sign(message []byte) ([]byte, error) {
	if k.signErr != nil {
		return nil, k.signErr
	}
	digest := sha256.Sum256(message)

	signature, err := ecdsa.SignASN1(rand.Reader, k.privateKey, digest[:])
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}

	return signature, nil
}

func ed25519KeyInfo(t *testing.T) []byte {
	t.Helper()
	keyInfo, err := base64.StdEncoding.DecodeString(
		"MCowBQYDK2VwAyEAGb9ECWmEzf6FQbrBZ9w7lshQhqowtrbLDFw4rXAxZuE=",
	)
	if err != nil {
		t.Fatal(err)
	}

	return keyInfo
}

func signerOver(keys *fakeKeys) statementsigning.Signer {
	return statementsigning.New(keys, android.New(nil), quietObserver{})
}

func parsedBundle(t *testing.T, encoding []byte) *verifier.Bundle {
	t.Helper()
	bundle, err := verifier.ParseBundle(encoding)
	if err != nil {
		t.Fatalf("bundle %s: %v", encoding, err)
	}

	return bundle
}

type quietObserver struct{}

func (quietObserver) KeyUnavailable(context.Context, error)        {}
func (quietObserver) SignatureRefused(context.Context, error)      {}
func (quietObserver) KeyEvidenceUnreadable(context.Context, error) {}
func (quietObserver) TimeStampUnavailable(context.Context, error)  {}
