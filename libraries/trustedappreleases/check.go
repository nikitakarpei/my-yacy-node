package trustedappreleases

import (
	"crypto/sha256"
	"fmt"
)

type ProvenanceVerifier interface {
	ProvenanceOf(signedBundle []byte, subject [sha256.Size]byte) (Provenance, error)
}

type Check struct {
	verifier ProvenanceVerifier
	workflow ReleaseWorkflow
}

type Verdict struct {
	Reason  error
	Release TrustedRelease
	Trusted bool
}

func New(workflow ReleaseWorkflow, verifier ProvenanceVerifier) Check {
	return Check{verifier: verifier, workflow: workflow}
}

func (c Check) VerdictOn(release Release) Verdict {
	if reason := c.workflow.reasonAgainstRelease(release); reason != nil {
		return untrusted(reason)
	}
	provenance, err := c.verifier.ProvenanceOf(release.SignedBundle, sha256.Sum256(release.Record))
	if err != nil {
		return untrusted(fmt.Errorf("verify the provenance of the release record: %w", err))
	}
	if reason := c.workflow.reasonAgainstProvenance(provenance, release); reason != nil {
		return untrusted(reason)
	}
	trustedRelease, err := trustedReleaseFrom(release.Tag, release.Record)
	if err != nil {
		return untrusted(err)
	}

	return Verdict{Trusted: true, Release: trustedRelease}
}

func untrusted(reason error) Verdict {
	return Verdict{Reason: reason}
}
