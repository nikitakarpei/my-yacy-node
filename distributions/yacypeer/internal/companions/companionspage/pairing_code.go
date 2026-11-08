package companionspage

import (
	"fmt"
	"html/template"
	"strings"

	"rsc.io/qr"
)

const quietZone = 4

func pairingCodeOf(link string) template.HTML {
	code, err := qr.Encode(link, qr.M)
	if err != nil {
		return ""
	}
	side := code.Size + 2*quietZone
	var path strings.Builder
	for y := range code.Size {
		for x := 0; x < code.Size; x++ {
			if !code.Black(x, y) {
				continue
			}
			run := 1
			for code.Black(x+run, y) {
				run++
			}
			fmt.Fprintf(&path, "M%d %dh%dv1h-%dz", x+quietZone, y+quietZone, run, run)
			x += run - 1
		}
	}
	return template.HTML(fmt.Sprintf( //nolint:gosec // G203: built from numbers only.
		`<svg viewBox="0 0 %d %d" role="img" aria-label="Pairing code" `+
			`shape-rendering="crispEdges"><rect width="%d" height="%d" fill="#fff"/>`+
			`<path fill="#000" d="%s"/></svg>`,
		side, side, side, side, path.String(),
	))
}
