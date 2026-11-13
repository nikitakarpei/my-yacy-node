// Package companionreleases names the YaCy companion app and the release workflow whose releases
// peers trust.
package companionreleases

const (
	AppPackage     = "io.github.nikitakarpei.yacypeercompanion"
	RepositoryID   = "1273925787"
	OwnerID        = "205461907"
	WorkflowPath   = ".github/workflows/release-companion-app.yml"
	TagPattern     = "yacypeercompanion-v*"
	DenialsAddress = "https://raw.githubusercontent.com/nikitakarpei/my-yacy-node/main/" +
		"distributions/yacypeercompanion/denied-releases.json"
)
