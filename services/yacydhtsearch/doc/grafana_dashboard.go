// Package yacydhtsearchdoc holds the operator documents of yacydhtsearch that code reads,
// such as its Grafana dashboard.
package yacydhtsearchdoc

import _ "embed"

//go:embed grafana-dashboard.json
var GrafanaDashboard []byte
