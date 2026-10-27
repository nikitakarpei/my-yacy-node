// Package yacycrawlerdoc holds the operator documents of yacycrawler that code reads,
// such as its Grafana dashboard.
package yacycrawlerdoc

import _ "embed"

//go:embed grafana-dashboard.json
var GrafanaDashboard []byte
