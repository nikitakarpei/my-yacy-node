package trustedappreleases

type Release struct {
	Tag          string
	Commit       string
	Record       []byte
	SignedBundle []byte
	Immutable    bool
}
