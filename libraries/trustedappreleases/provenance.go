package trustedappreleases

type Provenance struct {
	Workflow     string
	Repository   string
	RepositoryID string
	OwnerID      string
	Ref          string
	Commit       string
	Runner       string
	Visibility   string
}
