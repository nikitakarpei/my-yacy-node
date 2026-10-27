// Package pageshell gives the web pages of the peer one look and one way between them: the
// stylesheet that each page links before its own, and the tab bar at the top of each page.
package pageshell

import (
	"embed"
	"net/http"
)

const StylesheetPath = "/static/pageshell.css"

//go:embed static
var staticFiles embed.FS

func Stylesheet() http.Handler {
	return http.FileServerFS(staticFiles)
}
