// Package yacynodedoc holds the operator documents of yacynode that code reads,
// such as its Grafana dashboard.
package yacynodedoc

import _ "embed"

//go:embed grafana-dashboard.json
var GrafanaDashboard []byte
