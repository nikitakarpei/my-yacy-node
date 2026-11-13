// Package companionapp names the YaCy companion app, the time-stamp authority that its statements
// use, and the release workflow whose releases peers trust.
package companionapp

const (
	Package               = "io.github.nikitakarpei.yacypeercompanion"
	TimeStampAuthority    = "freetsa"
	RepositoryID          = "1273925787"
	OwnerID               = "205461907"
	ReleaseWorkflowPath   = ".github/workflows/release-companion-app.yml"
	ReleaseTagPattern     = "yacypeercompanion-v*"
	DeniedReleasesAddress = "https://raw.githubusercontent.com/nikitakarpei/my-yacy-node/main/" +
		"distributions/yacypeercompanion/denied-releases.json"
)
