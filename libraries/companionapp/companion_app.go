// Package companionapp names the YaCy companion app and the release workflow whose releases
// peers trust.
package companionapp

const (
	Package               = "io.github.nikitakarpei.yacypeercompanion"
	RepositoryID          = "1273925787"
	OwnerID               = "205461907"
	ReleaseWorkflowPath   = ".github/workflows/release-companion-app.yml"
	ReleaseTagPattern     = "yacypeercompanion-v*"
	DeniedReleasesAddress = "https://raw.githubusercontent.com/nikitakarpei/my-yacy-node/main/" +
		"distributions/yacypeercompanion/denied-releases.json"
)
