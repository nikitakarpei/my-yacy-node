package trustedappreleases

import "crypto/sha256"

type ProvenanceVerifier interface {
	ProvenanceOf(signedBundle []byte, subject [sha256.Size]byte) (Provenance, error)
}

type Provenance struct {
	WorkflowAddress   string
	RepositoryAddress string
	RepositoryID      string
	OwnerID           string
	Ref               string
	Commit            string
	Runner            string
	Visibility        string
}
