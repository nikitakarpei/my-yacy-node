package contentformatgraph

import (
	"fmt"
)

type Graph struct {
	byTargetFormat map[Format][]Derivation
}

func New(derivations []Derivation) Graph {
	byTargetFormat := make(
		map[Format][]Derivation,
		len(derivations),
	)
	for _, derivation := range derivations {
		byTargetFormat[derivation.TargetFormat()] = append(
			byTargetFormat[derivation.TargetFormat()],
			derivation,
		)
	}
	return Graph{byTargetFormat: byTargetFormat}
}

func (g Graph) Validate(targetFormats []Format) error {
	reachable := map[Format]bool{}
	var require func(Format) error
	require = func(format Format) error {
		if format == FormatDocumentHTML || reachable[format] {
			return nil
		}
		candidates, ok := g.byTargetFormat[format]
		if !ok {
			return fmt.Errorf("%s content is read but no derivation produces it", format)
		}
		reachable[format] = true
		for _, candidate := range candidates {
			if err := require(candidate.SourceFormat()); err != nil {
				return err
			}
		}
		return nil
	}
	for _, format := range targetFormats {
		if err := require(format); err != nil {
			return err
		}
	}
	return nil
}

func (g Graph) Resolver(
	pageURL string,
	format Format,
	body []byte,
) *Resolver {
	return &Resolver{
		pageURL:      pageURL,
		graph:        g,
		contents:     map[Format][]byte{format: body},
		unresolvable: make(map[Format]bool),
		resolving:    make(map[Format]bool),
	}
}
