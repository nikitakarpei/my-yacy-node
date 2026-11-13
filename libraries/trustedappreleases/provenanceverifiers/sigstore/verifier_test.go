package sigstore_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/root"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases/provenanceverifiers/sigstore"
)

var errNoTrustedRoot = errors.New("no trusted root")

type fixedRoot struct {
	trustedRoot root.TrustedMaterial
}

func (f fixedRoot) TrustedRoot() (root.TrustedMaterial, error) {
	if f.trustedRoot == nil {
		return nil, errNoTrustedRoot
	}

	return f.trustedRoot, nil
}

const yacypeerBinaryDigest = "e9a48e91adbe451cb3c2a01b7529e4c068accc5479287e5bb68e444367343e2a"

func TestAnAttestedSubjectHasTheProvenanceOfItsWorkflowRun(t *testing.T) {
	provenance, err := verifier(t).ProvenanceOf(attestation(t), yacypeerBinary(t))
	if err != nil {
		t.Fatalf("provenance: %v", err)
	}

	want := trustedappreleases.Provenance{
		WorkflowAddress:   "https://github.com/nikitakarpei/my-yacy-node/.github/workflows/build-yacypeer.yml@refs/heads/main",
		RepositoryAddress: "https://github.com/nikitakarpei/my-yacy-node",
		RepositoryID:      "1273925787",
		OwnerID:           "205461907",
		Ref:               "refs/heads/main",
		Commit:            "ada4d0e13a718d97ee957b8516a5f685d9816061",
		Runner:            "github-hosted",
		Visibility:        "public",
	}
	if provenance != want {
		t.Errorf("provenance = %+v, want %+v", provenance, want)
	}
}

func TestAnotherSubjectHasNoProvenance(t *testing.T) {
	otherRecord := sha256.Sum256([]byte("another release record"))

	_, err := verifier(t).ProvenanceOf(attestation(t), otherRecord)
	if err == nil {
		t.Error("an attestation of another subject verified")
	}
}

func TestAMalformedBundleHasNoProvenance(t *testing.T) {
	_, err := verifier(t).ProvenanceOf([]byte("{"), yacypeerBinary(t))
	if err == nil {
		t.Error("a malformed bundle verified")
	}
}

func TestABundleOutsideTheTrustedRootHasNoProvenance(t *testing.T) {
	emptyRoot, err := root.NewTrustedRoot(root.TrustedRootMediaType01, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = sigstore.New(fixedRoot{trustedRoot: emptyRoot}).
		ProvenanceOf(attestation(t), yacypeerBinary(t))
	if err == nil {
		t.Error("a bundle verified against an empty trusted root")
	}
}

func TestABundleWithoutATrustedRootHasNoProvenance(t *testing.T) {
	_, err := sigstore.New(fixedRoot{}).
		ProvenanceOf(attestation(t), yacypeerBinary(t))

	if !errors.Is(err, errNoTrustedRoot) {
		t.Errorf("error = %v, want %v", err, errNoTrustedRoot)
	}
}

func verifier(t *testing.T) sigstore.Verifier {
	t.Helper()
	trustedRoot, err := root.NewTrustedRootFromJSON(testdata(t, "trusted_root.json"))
	if err != nil {
		t.Fatal(err)
	}

	return sigstore.New(fixedRoot{trustedRoot: trustedRoot})
}

func attestation(t *testing.T) []byte {
	t.Helper()

	return testdata(t, "yacypeer-linux-amd64.sigstore.json")
}

func testdata(t *testing.T, name string) []byte {
	t.Helper()
	content, err := fs.ReadFile(os.DirFS("testdata"), name)
	if err != nil {
		t.Fatal(err)
	}

	return content
}

func yacypeerBinary(t *testing.T) [sha256.Size]byte {
	t.Helper()
	decoded, err := hex.DecodeString(yacypeerBinaryDigest)
	if err != nil || len(decoded) != sha256.Size {
		t.Fatalf("digest %q: %v", yacypeerBinaryDigest, err)
	}

	return [sha256.Size]byte(decoded)
}
