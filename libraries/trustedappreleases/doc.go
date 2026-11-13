// Package trustedappreleases tells which releases of an Android app peers can trust. A release
// is trusted when it is immutable, its tag matches the release workflow, and the verified
// provenance of its release record names that workflow of that repository at that tag and its
// commit. A trusted release gives the signing certificate and the versionCode of the app.
package trustedappreleases
