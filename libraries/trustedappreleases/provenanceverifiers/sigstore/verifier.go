// Package sigstore verifies the provenance that GitHub Actions attests through Sigstore. A
// bundle verifies when its certificate chains to the trusted root, has a certificate timestamp
// of a log, is in the transparency log, comes from the GitHub Actions token issuer, and its
// statement has the subject. The trusted root is taken again for each bundle. The provenance is
// the claims of the certificate about the run.
package sigstore

import (
	"crypto/sha256"
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

const githubActionsIssuer = "https://token.actions.githubusercontent.com"

type TrustedRoots interface {
	TrustedRoot() (root.TrustedMaterial, error)
}

type Verifier struct {
	trustedRoots TrustedRoots
}

func New(trustedRoots TrustedRoots) Verifier {
	return Verifier{trustedRoots: trustedRoots}
}

func (v Verifier) ProvenanceOf(
	signedBundle []byte,
	subject [sha256.Size]byte,
) (trustedappreleases.Provenance, error) {
	var parsedBundle bundle.Bundle
	if err := parsedBundle.UnmarshalJSON(signedBundle); err != nil {
		return trustedappreleases.Provenance{}, fmt.Errorf("read the bundle: %w", err)
	}
	summary, err := v.verifiedCertificateOf(&parsedBundle, subject)
	if err != nil {
		return trustedappreleases.Provenance{}, err
	}

	return provenanceFrom(summary), nil
}

func (v Verifier) verifiedCertificateOf(
	parsedBundle *bundle.Bundle,
	subject [sha256.Size]byte,
) (*certificate.Summary, error) {
	trustedRoot, err := v.trustedRoots.TrustedRoot()
	if err != nil {
		return nil, fmt.Errorf("take the Sigstore trusted root: %w", err)
	}
	bundleVerifier, err := verify.NewVerifier(trustedRoot,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1),
	)
	if err != nil {
		return nil, fmt.Errorf("make the bundle verifier: %w", err)
	}
	issuerIdentity, err := verify.NewShortCertificateIdentity(githubActionsIssuer, "", "", ".+")
	if err != nil {
		return nil, fmt.Errorf("name the certificate issuer: %w", err)
	}
	result, err := bundleVerifier.Verify(parsedBundle, verify.NewPolicy(
		verify.WithArtifactDigest("sha256", subject[:]),
		verify.WithCertificateIdentity(issuerIdentity),
	))
	if err != nil {
		return nil, fmt.Errorf("verify the bundle: %w", err)
	}

	return result.Signature.Certificate, nil
}

func provenanceFrom(summary *certificate.Summary) trustedappreleases.Provenance {
	return trustedappreleases.Provenance{
		WorkflowAddress:   summary.BuildSignerURI,
		RepositoryAddress: summary.SourceRepositoryURI,
		RepositoryID:      summary.SourceRepositoryIdentifier,
		OwnerID:           summary.SourceRepositoryOwnerIdentifier,
		Ref:               summary.SourceRepositoryRef,
		Commit:            summary.SourceRepositoryDigest,
		Runner:            summary.RunnerEnvironment,
		Visibility:        summary.SourceRepositoryVisibilityAtSigning,
	}
}
