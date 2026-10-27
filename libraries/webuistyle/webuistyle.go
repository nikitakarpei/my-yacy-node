// Package webuistyle gives the web UIs of the services one look and one way between them: the
// stylesheet that each page links before its own, and the tab bar at the top of each page.
package webuistyle

import (
	"embed"
	"net/http"
)

const StylesheetPath = "/static/webuistyle.css"

//go:embed static
var staticFiles embed.FS

func Stylesheet() http.Handler {
	return http.FileServerFS(staticFiles)
}
